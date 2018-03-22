package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	inFlightGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "in_flight_requests",
		Help: "Inflight requests being served.",
	})

	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total HTTP requests processed.",
		},
		[]string{"code", "method"},
	)

	requestDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "request_duration_seconds",
			Help: "HTTP request latencies in seconds.",
		},
		[]string{},
	)

	responseSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "response_size_bytes",
			Help: "HTTP response sizes in bytes.",
		},
		[]string{},
	)

	requestSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "request_size_bytes",
			Help: "HTTP request sizes in bytes.",
		},
		[]string{},
	)

	chartTotalGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "navigator",
			Name:      "total_charts_served",
			Help:      "Charts being served by index",
		},
		[]string{"index"},
	)

	chartVersionTotalGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "navigator",
			Name:      "total_chart_versions_served",
			Help:      "Chart versions being served by index",
		},
		[]string{"index"},
	)
)

func init() {
	prometheus.MustRegister(
		requestCounter,
		requestDuration,
		responseSize,
		requestSize,
		chartTotalGauge,
		chartVersionTotalGauge)
}

// MetricMiddleware wraps a http handler with prometheus metric instruments
func MetricMiddleware(handler http.Handler) http.Handler {
	return promhttp.InstrumentHandlerInFlight(inFlightGauge,
		promhttp.InstrumentHandlerCounter(requestCounter,
			promhttp.InstrumentHandlerDuration(requestDuration,
				promhttp.InstrumentHandlerResponseSize(responseSize,
					promhttp.InstrumentHandlerRequestSize(requestSize, handler),
				),
			),
		),
	)
}
