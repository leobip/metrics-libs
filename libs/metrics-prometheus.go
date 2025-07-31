// -*- coding: utf-8 -*-
// metrics-prometheus.go
package libs

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
	cluster = getClusterFromContext()

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
	health = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "health",
			Help: "Health status of the operator (1 = healthy, 0 = not healthy)",
		},
		[]string{"namespace", "cluster"},
	)
	memoryUsageBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Memory usage in bytes of the operator",
		},
		[]string{"namespace", "cluster"},
	)
	cpuUsageCores = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cpu_usage_cores",
			Help: "CPU usage in cores of the operator",
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
		prometheus.MustRegister(health)
		prometheus.MustRegister(memoryUsageBytes)
		prometheus.MustRegister(cpuUsageCores)

		// Initialize health to 1 (healthy)
		health.WithLabelValues(namespace, cluster).Set(1)

		// Collect initial pod metrics
		mem := getMemoryUsageBytes()
		memoryUsageBytes.WithLabelValues(namespace, cluster).Set(mem)

		cpu := getCPUUsageCores()
		cpuUsageCores.WithLabelValues(namespace, cluster).Set(cpu)

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
