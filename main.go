package main

import (
	"log"

	metricslibs "github.com/leobip/metrics-libs/lib" // Reemplaza con la ruta correcta a tu paquete de métricas
)

func main() {
	err := metricslibs.StartPrometheusMetrics()
	if err != nil {
		log.Fatalf("could not start metrics server: %v", err)
	}

	log.Println("📊 Prometheus metrics available at :2112/metrics")
	select {} // bloquea indefinidamente
}
