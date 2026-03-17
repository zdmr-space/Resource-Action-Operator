package engine

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	metricsInit sync.Once

	httpRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "resource_action_operator_http_runs_total",
			Help: "Total number of ResourceAction HTTP execution runs by result.",
		},
		[]string{"result"},
	)

	httpActionsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "resource_action_operator_http_actions_total",
			Help: "Total number of HTTP actions executed.",
		},
	)

	httpAttemptsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "resource_action_operator_http_attempts_total",
			Help: "Total number of HTTP attempts across all actions.",
		},
	)

	httpRetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "resource_action_operator_http_retries_total",
			Help: "Total number of HTTP retries by retry type.",
		},
		[]string{"type"},
	)

	httpBackoffSecondsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "resource_action_operator_http_backoff_seconds_total",
			Help: "Accumulated HTTP backoff time in seconds.",
		},
	)

	httpDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "resource_action_operator_http_duration_seconds",
			Help:    "Distribution of HTTP execution duration per ResourceAction run.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"result"},
	)

	httpLastStatusTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "resource_action_operator_http_last_status_total",
			Help: "Count of observed final HTTP status classes.",
		},
		[]string{"class"},
	)
)

func initEngineMetrics() {
	metricsInit.Do(func() {
		ctrlmetrics.Registry.MustRegister(
			httpRunsTotal,
			httpActionsTotal,
			httpAttemptsTotal,
			httpRetriesTotal,
			httpBackoffSecondsTotal,
			httpDurationSeconds,
			httpLastStatusTotal,
		)
	})
}

func statusClass(status int) string {
	if status < 100 || status > 999 {
		return "unknown"
	}
	return fmt.Sprintf("%dxx", status/100)
}

func observeHTTPExecution(result string, recordMetrics HTTPExecutionRecordMetrics) {
	initEngineMetrics()

	httpRunsTotal.WithLabelValues(result).Inc()
	httpActionsTotal.Add(float64(recordMetrics.ActionCount))
	httpAttemptsTotal.Add(float64(recordMetrics.Attempts))
	httpRetriesTotal.WithLabelValues("network").Add(float64(recordMetrics.NetworkRetryCount))
	httpRetriesTotal.WithLabelValues("status").Add(float64(recordMetrics.StatusRetryCount))
	httpBackoffSecondsTotal.Add(float64(recordMetrics.BackoffMillis) / 1000.0)
	httpDurationSeconds.WithLabelValues(result).Observe(float64(recordMetrics.DurationMillis) / 1000.0)
	if recordMetrics.LastHTTPStatus > 0 {
		httpLastStatusTotal.WithLabelValues(statusClass(recordMetrics.LastHTTPStatus)).Inc()
	}
}

type HTTPExecutionRecordMetrics struct {
	ActionCount       int
	Attempts          int
	NetworkRetryCount int
	StatusRetryCount  int
	BackoffMillis     int64
	DurationMillis    int64
	LastHTTPStatus    int
}
