# 📊 Golang Library to get / process / send Metrics from Kubernetes Operator to Prometheus/Grafana & Kafka

A reusable Go library to expose Prometheus metrics from Kubernetes operators or services.

## Features

- Minimal setup
- Cluster and namespace-aware metrics
- Designed for use in operators
- Extensible (Kafka support, more metrics)

## 1.2. Estructura inicial de carpetas y archivos

```bash
metrics-libs/
├── go.mod
├── lib/
│   ├── metrics-common.go
│   ├── metrics-prometheus.go
│   └── metrics-kafka.go     # (opcional si lo implementas más adelante)
├── main.go                  # Para pruebas locales
└── README.md
└── hack/
    └── boilerplate.go.txt     # Cabecera de licencia si la necesitas
```

```bash
touch main.go metrics-common.go metrics-prometheus.go metrics-kafka.go
mkdir hack
touch hack/boilerplate.go.txt
```

### Add Dependencies

- Prometheus and basics libraries for Go

```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
go get k8s.io/client-go@latest
go get sigs.k8s.io/controller-runtime@latest

```

### main.go

```go
    package main

    import (
        "log"
        "net/http"
        "github.com/leobip/metrics-libs"
    )

    func main() {
        err := metricslibs.StartPrometheusMetrics()
        if err != nil {
            log.Fatalf("could not start metrics server: %v", err)
        }

        log.Println("📊 Prometheus metrics available at :2112/metrics")
        select {} // bloquea indefinidamente
    }
```


