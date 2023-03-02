package handlers

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"notary-admission/pkg/admissioncontroller/pods"
	"notary-admission/pkg/metrics"
	"notary-admission/pkg/model"
)

// NewTlsServer creates and return a http.Server with a mux that handles endpoints over TLS
func NewTlsServer(port string) *http.Server {
	phm := metrics.InitPrometheusHttpMetric(model.ServerConfig.Prometheus.Name,
		prometheus.LinearBuckets(model.ServerConfig.Prometheus.Start,
			model.ServerConfig.Prometheus.Width, model.ServerConfig.Prometheus.Count))

	// Instances hooks
	podsValidation := pods.NewValidationHook()

	// Routers
	ah := newAdmissionHandler()
	mux := http.NewServeMux()
	mux.Handle(model.ServerConfig.Network.Endpoints.PodValidation,
		phm.WrapHandler("pod-validator", ah.Serve(podsValidation)))

	return &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}
}

// NewServer creates and return a http.Server
func NewServer(port string) *http.Server {
	// Routers
	c := controller{}
	mux := http.NewServeMux()
	mux.Handle(model.ServerConfig.Network.Endpoints.Metrics, promhttp.Handler())
	mux.Handle(model.ServerConfig.Network.Endpoints.Health, c.healthz())

	return &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}
}
