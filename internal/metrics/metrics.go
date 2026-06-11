package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry is the custom Prometheus registry used by recall. Using a custom
// registry (instead of prometheus.DefaultRegisterer) keeps the server's
// metrics isolated from anything a transitive dependency might register.
var Registry = prometheus.NewRegistry()

// HTTPRequestsTotal counts HTTP requests served. The route label must be the
// route pattern (e.g. "/documents/{id}"), never the resolved path, to keep
// cardinality bounded.
var HTTPRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "recall_http_requests_total",
		Help: "Total number of HTTP requests served, labelled by method, route pattern, and status code.",
	},
	[]string{"method", "route", "status"},
)

// HTTPRequestDuration measures HTTP request latency in seconds. DefBuckets is
// the client_golang default (5ms .. 10s).
var HTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "recall_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds, labelled by method and route pattern.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "route"},
)

// DocumentsUploadedTotal counts successful document uploads. Incremented only
// after the file is persisted on disk and the row is inserted in Postgres.
var DocumentsUploadedTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "recall_documents_uploaded_total",
		Help: "Total number of documents successfully uploaded.",
	},
)

func init() {
	Registry.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		DocumentsUploadedTotal,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

// Handler returns the http.Handler that serves the /metrics endpoint backed
// by the recall custom registry.
func Handler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{Registry: Registry})
}
