// -*- coding: utf-8 -*-
// metrics-prometheus.go
package metricslibs

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Common Labels for metrics
var (
	namespace = getNamespace()
	cluster   = getClusterFromContext()

	once sync.Once

	// Prometheus metrics Definition
	reconcileCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reconcile_count",
			Help: "Number of reconciliations performed",
		},
		[]string{"namespace", "cluster"},
	)
	reconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reconcile_errors",
			Help: "Number reconciliation errors",
		},
		[]string{"namespace", "cluster"},
	)
)

// StartPrometheusMetrics initializes and starts the Prometheus metrics server on /metrics
func StartPrometheusMetrics() error {
	var err error
	once.Do(func() {
		// Register metrics
		prometheus.MustRegister(reconcileCount)
		prometheus.MustRegister(reconcileErrors)

		// Valores de ejemplo iniciales
		reconcileCount.WithLabelValues(namespace, cluster).Add(3)
		reconcileErrors.WithLabelValues(namespace, cluster).Add(1)

		// Get the port by environment variable or default to 2112
		port := os.Getenv("METRICS_PORT")
		if port == "" {
			port = "2112"
		}
		addr := fmt.Sprintf(":%s", port)
		log.Printf("Starting Prometheus metrics server on %s\n", addr)
		// Start HTTP server
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			log.Printf("Prometheus metrics available at %s/metrics\n", addr)
			if e := http.ListenAndServe(addr, nil); e != nil {
				log.Printf("Error starting metrics server: %v\n", e)
				err = e
			}
		}()
	})
	return err
}
