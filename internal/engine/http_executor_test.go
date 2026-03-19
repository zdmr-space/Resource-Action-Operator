package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	opsv1alpha1 "de.yusaozdemir.resource-action-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHTTPExecutorExecuteWithMetrics_StatusRetry(t *testing.T) {
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 3 {
			http.Error(w, "retry", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	exec := NewHTTPExecutor(fake.NewClientBuilder().Build())
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "demo",
				"namespace": "default",
				"uid":       "u1",
			},
		},
	}

	metrics, err := exec.ExecuteWithMetrics(context.Background(), opsv1alpha1.ActionSpec{
		Type:           "http",
		Method:         "POST",
		URL:            srv.URL,
		URLPolicy:      &opsv1alpha1.URLPolicySpec{AllowUnsafeLocalTargets: true},
		ExpectedStatus: "^2..$",
		Timeout:        "2s",
		Retry: &opsv1alpha1.RetrySpec{
			MaxAttempts:   3,
			Backoff:       "1ms",
			MaxBackoff:    "2ms",
			RetryOnStatus: []int{500},
		},
	}, "default", obj, map[string]string{"X-Test": "1"})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if metrics.Attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", metrics.Attempts)
	}
	if metrics.StatusRetryCount != 2 {
		t.Fatalf("expected 2 status retries, got %d", metrics.StatusRetryCount)
	}
	if metrics.NetworkRetryCount != 0 {
		t.Fatalf("expected 0 network retries, got %d", metrics.NetworkRetryCount)
	}
	if metrics.StatusCode != http.StatusOK {
		t.Fatalf("expected final status 200, got %d", metrics.StatusCode)
	}
	if metrics.DurationMillis < 0 {
		t.Fatalf("expected non-negative duration, got %d", metrics.DurationMillis)
	}
}

func TestValidateTargetURL_DefaultBlocked(t *testing.T) {
	err := validateTargetURL("http://127.0.0.1:8080/hook", nil)
	if err == nil {
		t.Fatalf("expected localhost/IP safety policy error, got nil")
	}
}

func TestValidateTargetURL_AllowlistAndBlocklist(t *testing.T) {
	policy := &opsv1alpha1.URLPolicySpec{
		AllowedHostRegex: []string{`^api\.example\.com$`},
		BlockedHostRegex: []string{`^blocked\.example\.com$`},
	}

	if err := validateTargetURL("https://api.example.com/hook", policy); err != nil {
		t.Fatalf("expected allowed host to pass, got error: %v", err)
	}

	if err := validateTargetURL("https://blocked.example.com/hook", policy); err == nil {
		t.Fatalf("expected blocked host to fail, got nil")
	}

	if err := validateTargetURL("https://other.example.com/hook", policy); err == nil {
		t.Fatalf("expected non-allowlisted host to fail, got nil")
	}
}

func TestValidateTargetURL_AllowUnsafeLocalTargets(t *testing.T) {
	policy := &opsv1alpha1.URLPolicySpec{AllowUnsafeLocalTargets: true}
	if err := validateTargetURL("http://127.0.0.1:8080/hook", policy); err != nil {
		t.Fatalf("expected localhost to be allowed when explicitly opted in, got error: %v", err)
	}
}
