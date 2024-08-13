package consulkv

import (
	"sync"

	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"

	"github.com/prometheus/client_golang/prometheus"
)

var metricsSubsystem = "consulkv"

var metricsPluginErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: metricsSubsystem,
	Name:      "plugin_errors_total",
	Help:      "Count the amount of errors within the plugin.",
}, []string{"error"})

func IncrementMetricsPluginErrorsTotal(err string) {
	metricsPluginErrorsTotal.WithLabelValues(err).Inc()
}

var metricsConsulConfigUpdatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: metricsSubsystem,
	Name:      "consul_config_updated_total",
	Help:      "Count the amount of times the config was updated from the Consul key/value.",
}, []string{"error"})

func IncrementMetricsConsulConfigUpdatedTotal(err string) {
	metricsConsulConfigUpdatedTotal.WithLabelValues(err).Inc()
}

var metricsConsulRequestDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: plugin.Namespace,
	Subsystem: metricsSubsystem,
	Name:      "consul_request_duration_seconds",
	Help:      "Histogram of the time (in seconds) each request to Consul took.",
	Buckets:   []float64{.001, .002, .005, .01, .02, .05, .1, .2, .5, 1},
}, []string{"status"})

func IncrementMetricsConsulRequestDurationSeconds(status string, duration float64) {
	metricsConsulRequestDurationSeconds.WithLabelValues(status).Observe(duration)
}

var metricsQueryRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: metricsSubsystem,
	Name:      "query_requests_total",
	Help:      "Count the amount of queries received as request by the plugin.",
}, []string{"zone", "type"})

func IncrementMetricsQueryRequestsTotal(zone string, qtype uint16) {
	t := dns.TypeToString[qtype]
	metricsQueryRequestsTotal.WithLabelValues(dns.Fqdn(zone), t).Inc()
}

var metricsQueryResponsesSuccessfulTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: metricsSubsystem,
	Name:      "query_responses_successful_total",
	Help:      "Count the amount of successful queries handled and responded to by the plugin.",
}, []string{"zone", "type"})

func IncrementMetricsResponsesSuccessfulTotal(zone string, qtype uint16) {
	t := dns.TypeToString[qtype]
	metricsQueryResponsesSuccessfulTotal.WithLabelValues(dns.Fqdn(zone), t).Inc()
}

var metricsQueryResponsesFailedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: metricsSubsystem,
	Name:      "query_responses_failed_total",
	Help:      "Count the amount of failed queries handled by the plugin.",
}, []string{"zone", "type", "error"})

func IncrementMetricsResponsesFailedTotal(zone string, qtype uint16, err string) {
	t := dns.TypeToString[qtype]
	metricsQueryResponsesFailedTotal.WithLabelValues(dns.Fqdn(zone), t, err).Inc()
}

var _ sync.Once
