package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type EndpointLabelCollector interface {
	EndpointLabelForURI(uri string) string
}

func NewDefaultEndpointLabelCollector() EndpointLabelCollector {
	return defaultEndpointLabelCollector{}
}

type defaultEndpointLabelCollector struct {
}

func (d defaultEndpointLabelCollector) EndpointLabelForURI(uri string) string {
	return uri
}

func NewPrometheusMetricsHandler(endpointLabelCollector EndpointLabelCollector) (PrometheusMetricsHandler, error) {
	handler := &prometheusMetricsHandler{
		endpointLabelCollector: endpointLabelCollector,
	}
	if err := handler.initCommonMetrics(); err != nil {
		return handler, err
	}
	return handler, nil
}

type PrometheusMetricsHandler interface {
	AddMetricsHandler(router *mux.Router, metricsEndpoint string)
	AddCommonMetricsMiddleware(router *mux.Router)
}

type prometheusMetricsHandler struct {
	endpointLabelCollector EndpointLabelCollector

	latencyHistogram *prometheus.HistogramVec
	requestCounter   *prometheus.CounterVec
}

func (p *prometheusMetricsHandler) AddMetricsHandler(router *mux.Router, endpoint string) {
	router.Handle(endpoint, promhttp.Handler())
}

func (p *prometheusMetricsHandler) AddCommonMetricsMiddleware(router *mux.Router) {
	router.Use(p.prometheusMiddleware())
}

func (p *prometheusMetricsHandler) prometheusMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			code := http.StatusOK

			defer func() {
				httpDuration := time.Since(start)
				endpoint := p.endpointLabelCollector.EndpointLabelForURI(req.RequestURI)
				labels := []string{endpoint, req.Method, fmt.Sprintf("%d", code)}
				p.latencyHistogram.WithLabelValues(labels...).Observe(httpDuration.Seconds())
				p.requestCounter.WithLabelValues(labels...).Inc()
			}()

			rw := &responseWriter{w, http.StatusOK}
			next.ServeHTTP(rw, req)
			code = rw.statusCode
		})
	}
}

func (p *prometheusMetricsHandler) initCommonMetrics() error {
	labelNames := []string{"endpoint", "method", "status"}

	p.latencyHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "app_request_latency_seconds",
		Help:    "Application Request Latency",
		Buckets: prometheus.DefBuckets,
	}, labelNames)
	err := prometheus.Register(p.latencyHistogram)
	if err != nil {
		return err
	}

	p.requestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "app_request_count",
		Help: "Application Request Count",
	}, labelNames)
	err = prometheus.Register(p.requestCounter)
	if err != nil {
		return err
	}

	return nil
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
