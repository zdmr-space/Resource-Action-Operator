package engine

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	opsv1alpha1 "de.yusaozdemir.resource-action-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type TemplateContext struct {
	Event    string                 `json:"event"`
	ActionID string                 `json:"actionId,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

type HTTPExecutor struct {
	k8s client.Client
	rng *rand.Rand
}

type HTTPExecutionMetrics struct {
	Attempts          int
	StatusCode        int
	NetworkRetryCount int
	StatusRetryCount  int
	BackoffMillis     int64
	DurationMillis    int64
}

func NewHTTPExecutor(k8s client.Client) *HTTPExecutor {
	return &HTTPExecutor{
		k8s: k8s,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (h *HTTPExecutor) Execute(
	ctx context.Context,
	action opsv1alpha1.ActionSpec,
	raNamespace string,
	obj *unstructured.Unstructured,
	headers map[string]string,
) error {
	_, err := h.ExecuteWithMetrics(ctx, action, raNamespace, obj, headers)
	return err
}

func (h *HTTPExecutor) ExecuteWithMetrics(
	ctx context.Context,
	action opsv1alpha1.ActionSpec,
	raNamespace string,
	obj *unstructured.Unstructured,
	headers map[string]string,
) (HTTPExecutionMetrics, error) {
	logger := log.FromContext(ctx)
	startedAt := time.Now()
	metrics := HTTPExecutionMetrics{}

	timeout := parseDurationDefault(action.Timeout, 10*time.Second)

	maxAttempts := 1
	backoffBase := 500 * time.Millisecond
	maxBackoff := 10 * time.Second
	retryOnNetwork := true
	retryOnStatus := map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true}

	if action.Retry != nil {
		if action.Retry.MaxAttempts > 0 {
			maxAttempts = action.Retry.MaxAttempts
		}
		backoffBase = parseDurationDefault(action.Retry.Backoff, backoffBase)
		maxBackoff = parseDurationDefault(action.Retry.MaxBackoff, maxBackoff)

		if action.Retry.RetryOnNetworkError != nil {
			retryOnNetwork = *action.Retry.RetryOnNetworkError
		}
		if len(action.Retry.RetryOnStatus) > 0 {
			retryOnStatus = map[int]bool{}
			for _, s := range action.Retry.RetryOnStatus {
				retryOnStatus[s] = true
			}
		}
	}

	transport, err := h.buildTransport(ctx, raNamespace, action.TLS)
	if err != nil {
		return metrics, err
	}

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	var bodyBytes []byte
	if action.Body != nil && action.Body.Template != "" {
		tpl, err := template.New("body").Parse(action.Body.Template)
		if err != nil {
			return metrics, err
		}

		var buf bytes.Buffer

		err = tpl.Execute(&buf, obj.Object)
		if err != nil {
			return metrics, err
		}

		bodyBytes = buf.Bytes()
	}

	method := action.Method
	if method == "" {
		method = "POST"
	}

	pattern := action.ExpectedStatus
	if pattern == "" {
		pattern = "^2..$"
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return metrics, fmt.Errorf("invalid expectedStatus regex: %w", err)
	}
	if err := validateTargetURL(action.URL, action.URLPolicy); err != nil {
		return metrics, err
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		metrics.Attempts = attempt

		var bodyReader io.Reader
		if len(bodyBytes) > 0 {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(reqCtx, method, action.URL, bodyReader)
		if err != nil {
			cancel()
			metrics.DurationMillis = time.Since(startedAt).Milliseconds()
			return metrics, err
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if len(bodyBytes) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := httpClient.Do(req)
		cancel()
		if err != nil {
			// network error?
			if retryOnNetwork && attempt < maxAttempts && isRetryableNetErr(err) {
				sleep := backoffSleep(h.rng, backoffBase, maxBackoff, attempt)
				metrics.NetworkRetryCount++
				metrics.BackoffMillis += sleep.Milliseconds()
				logger.Info("HTTP retry (network error)",
					"url", action.URL,
					"attempt", attempt,
					"sleep", sleep.String(),
					"error", err.Error(),
				)
				time.Sleep(sleep)
				continue
			}
			metrics.DurationMillis = time.Since(startedAt).Milliseconds()
			return metrics, err
		}

		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		metrics.StatusCode = resp.StatusCode

		logger.Info("HTTP action executed",
			"url", action.URL,
			"status", resp.StatusCode,
			"attempt", attempt,
			"response", string(respBody),
		)

		statusStr := strconv.Itoa(resp.StatusCode)
		if re.MatchString(statusStr) {
			metrics.DurationMillis = time.Since(startedAt).Milliseconds()
			return metrics, nil
		}

		// retry on configured status codes
		if retryOnStatus[resp.StatusCode] && attempt < maxAttempts {
			sleep := backoffSleep(h.rng, backoffBase, maxBackoff, attempt)
			metrics.StatusRetryCount++
			metrics.BackoffMillis += sleep.Milliseconds()
			logger.Info("HTTP retry (status)",
				"url", action.URL,
				"status", resp.StatusCode,
				"attempt", attempt,
				"sleep", sleep.String(),
			)
			time.Sleep(sleep)
			continue
		}

		// final error
		metrics.DurationMillis = time.Since(startedAt).Milliseconds()
		return metrics, fmt.Errorf("http call failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	metrics.DurationMillis = time.Since(startedAt).Milliseconds()
	return metrics, fmt.Errorf("http call failed after %d attempts", maxAttempts)
}

func (h *HTTPExecutor) buildTransport(ctx context.Context, raNamespace string, tlsSpec *opsv1alpha1.TLSSpec) (*http.Transport, error) {
	// base transport (keepalive)
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// No TLS config needed
	if tlsSpec == nil {
		// default TLS settings still apply for https via system roots
		return tr, nil
	}

	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: tlsSpec.InsecureSkipVerify,
	}

	if tlsSpec.ServerName != "" {
		cfg.ServerName = tlsSpec.ServerName
	}

	// CA from secret
	if tlsSpec.CaSecretRef != nil {
		var sec corev1.Secret
		if err := h.k8s.Get(ctx, client.ObjectKey{
			Name:      tlsSpec.CaSecretRef.Name,
			Namespace: raNamespace,
		}, &sec); err != nil {
			return nil, err
		}

		ca := sec.Data[tlsSpec.CaSecretRef.Key]
		if len(ca) == 0 {
			return nil, fmt.Errorf("caSecretRef %s/%s key %q empty", raNamespace, tlsSpec.CaSecretRef.Name, tlsSpec.CaSecretRef.Key)
		}

		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(ca); !ok {
			return nil, fmt.Errorf("failed to parse CA PEM from %s/%s", raNamespace, tlsSpec.CaSecretRef.Name)
		}
		cfg.RootCAs = pool
	}

	// mTLS client cert
	if tlsSpec.ClientCertSecretRef != nil {
		var sec corev1.Secret
		if err := h.k8s.Get(ctx, client.ObjectKey{
			Name:      tlsSpec.ClientCertSecretRef.Name,
			Namespace: raNamespace,
		}, &sec); err != nil {
			return nil, err
		}

		certPEM := sec.Data[tlsSpec.ClientCertSecretRef.CertKey]
		keyPEM := sec.Data[tlsSpec.ClientCertSecretRef.KeyKey]
		if len(certPEM) == 0 || len(keyPEM) == 0 {
			return nil, fmt.Errorf("clientCertSecretRef %s/%s missing cert/key", raNamespace, tlsSpec.ClientCertSecretRef.Name)
		}

		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	tr.TLSClientConfig = cfg
	return tr, nil
}

func parseDurationDefault(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

func backoffSleep(rng *rand.Rand, base, max time.Duration, attempt int) time.Duration {
	// exponential: base * 2^(attempt-1)
	mult := 1 << (attempt - 1)
	sleep := time.Duration(int64(base) * int64(mult))
	if sleep > max {
		sleep = max
	}

	// jitter: 0..25% of sleep
	jitterMax := int64(sleep) / 4
	if jitterMax > 0 {
		sleep += time.Duration(rng.Int63n(jitterMax))
	}
	return sleep
}

func isRetryableNetErr(err error) bool {
	// very pragmatic: timeout / connection resets
	if nerr, ok := err.(net.Error); ok {
		if nerr.Timeout() {
			return true
		}
	}
	// match common strings (safe-ish)
	msg := err.Error()
	re := regexp.MustCompile(`(?i)connection reset|broken pipe|EOF|i/o timeout|tls handshake timeout`)
	return re.MatchString(msg)
}

func validateTargetURL(rawURL string, policy *opsv1alpha1.URLPolicySpec) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid action URL: %w", err)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("invalid action URL: host is empty")
	}

	if (policy == nil || !policy.AllowUnsafeLocalTargets) && isDefaultBlockedHost(host) {
		return fmt.Errorf("action URL host %q is blocked by default safety policy", host)
	}

	if policy == nil {
		return nil
	}

	if len(policy.BlockedHostRegex) > 0 {
		blocked, err := matchAnyRegex(policy.BlockedHostRegex, host)
		if err != nil {
			return err
		}
		if blocked {
			return fmt.Errorf("action URL host %q is blocked by urlPolicy.blockedHostRegex", host)
		}
	}

	if len(policy.AllowedHostRegex) > 0 {
		allowed, err := matchAnyRegex(policy.AllowedHostRegex, host)
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("action URL host %q is not allowed by urlPolicy.allowedHostRegex", host)
		}
	}

	return nil
}

func matchAnyRegex(patterns []string, value string) (bool, error) {
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return false, fmt.Errorf("invalid urlPolicy regex %q: %w", p, err)
		}
		if re.MatchString(value) {
			return true, nil
		}
	}
	return false, nil
}

func isDefaultBlockedHost(host string) bool {
	if host == "localhost" ||
		host == "0.0.0.0" ||
		host == "127.0.0.1" ||
		host == "::1" ||
		host == "169.254.169.254" ||
		host == "metadata.google.internal" ||
		host == "metadata" {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate()
	}

	return false
}
