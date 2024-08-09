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

var metricsQueryRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: metricsSubsystem,
	Name:      "query_requests_total",
	Help:      "Count the amount of queries received as request by the plugin.",
}, []string{"zone", "type"})

func IncrementMetricsQueryRequestsTotal(zone string, qtype uint16) {
	t := dns.TypeToString[qtype]
	metricsQueryRequestsTotal.WithLabelValues(zone, t).Inc()
}

var metricsQueryResponsesSuccessfulTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: metricsSubsystem,
	Name:      "query_responses_successful_total",
	Help:      "Count the amount of successful queries handled and responded to by the plugin.",
}, []string{"zone", "type"})

func IncrementMetricsResponsesSuccessfulTotal(zone string, qtype uint16) {
	t := dns.TypeToString[qtype]
	metricsQueryResponsesSuccessfulTotal.WithLabelValues(zone, t).Inc()
}

var metricsQueryResponsesFailedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: plugin.Namespace,
	Subsystem: metricsSubsystem,
	Name:      "query_responses_failed_total",
	Help:      "Count the amount of failed queries handled by the plugin.",
}, []string{"zone", "type", "error"})

func IncrementMetricsResponsesFailedTotal(zone string, qtype uint16, err string) {
	t := dns.TypeToString[qtype]
	metricsQueryResponsesFailedTotal.WithLabelValues(zone, t, err).Inc()
}

var _ sync.Once
