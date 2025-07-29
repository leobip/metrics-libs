// metrics-common.go
package metricslibs

import (
	"os"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	OperatorHealth = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "operator_health",
		Help: "Health status of the operator (1=healthy, 0=unhealthy).",
	})
)

type Metric struct {
	Name  string
	Value any
	Type  string // "gauge", "counter", etc.
	/*
		 	Labels map[string]string
			Error  error
	*/
}

func getMetrics() []Metric {
	return []Metric{
		{"memory_usage_bytes", getMemoryUsageBytes(), "gauge"},
		{"cpu_usage_cores", getCPUUsageCores(), "gauge"},
		/* 		{"health", getHealthStatus(), "gauge"},
		   		{"reconcile_count", getReconcileCount(), "counter"},
		   		{"reconcile_errors", getReconcileErrors(), "counter"},
		   		{"last_successful_reconcile_time", getLastSuccessfulReconcileTime(), "gauge"}, */
		// ...
	}
}

func SetOperatorHealthy(healthy bool) {
	if healthy {
		OperatorHealth.Set(1)
	} else {
		OperatorHealth.Set(0)
	}
}

// Extrae el namespace del entorno o pone 'default'
func getNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return "default"
}

// Extrae el cluster desde KUBECONTEXT o usa 'local'
func getClusterFromContext() string {
	if ctx := os.Getenv("KUBECONTEXT"); ctx != "" {
		return ctx
	}
	return "local"
}

func getMemoryUsageBytes() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) // Memoria actualmente asignada
}

func getCPUUsageCores() float64 {
	return float64(runtime.NumGoroutine())
}
