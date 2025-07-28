# 📊 Golang Library to get / process / send Metrics from Kubernetes Operator to Prometheus/Grafana & Kafka

A reusable Go library to expose Prometheus metrics from Kubernetes operators or services.

## Features

- Minimal setup
- Cluster and namespace-aware metrics
- Designed for use in operators
- Extensible (Kafka support, more metrics)

## 📁 1.2. Initial Project Structure

```bash
metrics-libs/
❯ tree
.
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
├── libs
│   ├── metrics-common.go
│   └── metrics-prometheus.go
├── local-test
│   └── main.go
├── metrics-example
│   └── pods-sample.yaml
└── README.md
```

```bash
touch main.go metrics-common.go metrics-prometheus.go metrics-kafka.go
mkdir hack
touch hack/boilerplate.go.txt
```

### 🔗 Add Dependencies

- Install Prometheus and Kubernetes-related Go libraries:

```bash
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
go get k8s.io/client-go@latest
go get sigs.k8s.io/controller-runtime@latest

```

## 🚀 Deploy Example Workloads on Minikube for Metrics Testing

### 1.- Create a Test Namespace

- The metrics-example/ folder contains a manifest to deploy several sample workloads:
  - nginx-sample-1: A basic Nginx pod simulating web traffic.
  - busybox-sample: A minimal pod printing messages every 5 seconds.
  - nginx-deploy: A Deployment with 2 replicas, simulating a scalable service.
  - stress-cpu: A pod that generates CPU load for 5 minutes using stress.

- To Deploy
  - Create namespace
  - Apply the manifest

```bash
kubectl create namespace metrics-ex

kubectl apply -f metrics-example/pods-sample.yaml
```

- Verify status

```bash
kubectl get pods -n metrics-ex
kubectl apply -f metrics-example/pods-sample.yaml
```

- Example output:

```bash

NAME                            READY   STATUS    RESTARTS   AGE
busybox-sample                  1/1     Running   0          3m14s
nginx-deploy-77b7c4686c-2vqp8   1/1     Running   0          3m14s
nginx-deploy-77b7c4686c-drdsv   1/1     Running   0          3m14s
nginx-sample-1                  1/1     Running   0          3m14s
stress-cpu                      1/1     Running   0          7s


NAME           READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deploy   2/2     2            2           4m2s

```

## 🧪 Local Test

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
        select {}
    }
```

### Run the library locally

```bash
go run local-test/main.go

# You have to see this in the output
📊 Prometheus metrics available at :2112/metrics

```

### Verify that metrics are available

```bash
curl localhost:2112/metrics | grep reconcile

# Output
# HELP reconcile_count Number of reconciliations performed
# TYPE reconcile_count counter
reconcile_count{namespace="metrics-ex",cluster="minikube"} 3
```

## 🧪 Test With Operator Integration

### ⛔ Skipping Remote Download (Local Replace)

- If you don't want to donwload the library from remote repository
  - If you are developing locally both projects (operator & library), you can use areplace in the go.mod for the library repo url
  - In the last section, after the requireds, add this

```go
replace github.com/leobip/metrics-libs => ../metrics-libs
```

- Your import in the operator should look like this:

```go
import (
    metricslibs "github.com/leobip/metrics-libs/libs"
)
```

- Call the library from the operator
- main.go

```go
func main() {
    err := metricslibs.StartPrometheusMetrics()
    if err != nil {
        log.Fatalf("⚠️ could not start Prometheus metrics: %v", err)
    }

    // Continúa con la inicialización del manager y controlador
}
```

- ***NOTE: If use 📝 kubebuilder, assure to call StartPrometheusMetrics() before or inside main() and out of the reconciler, so the server work with it from the start.***

### 📦 Set Environment Variables (execute this commnands or use it in a vscode - launch.json )

```bash
export WATCH_NAMESPACE=metrics-ex
export POD_NAMESPACE=metrics-ex
export KUBECONTEXT=$(kubectl config current-context)
```

### ▶️ Run the Operator

```bash
❯ go run cmd/main.go

2025-07-27T15:03:42+02:00       INFO    setup   starting manager
2025/07/27 15:03:42 Starting Prometheus metrics server on :2112
2025-07-27T15:03:42+02:00       INFO    setup   📊 custom Prometheus metrics server started on :2112
2025/07/27 15:03:42 Prometheus metrics available at :2112/metrics
2025-07-27T15:03:42+02:00       INFO    starting server {"name": "health probe", "addr": "[::]:8081"}
2025-07-27T15:03:42+02:00       INFO    controller-runtime.metrics      Starting metrics server
2025-07-27T15:03:42+02:00       INFO    controller-runtime.metrics      Serving metrics server  {"bindAddress": ":8080", "secure": false}
2025-07-27T15:03:42+02:00       INFO    Starting EventSource    {"controller": "simple", "controllerGroup": "demo.demo.local", "controllerKind": "Simple", "source": "kind source: *v1.Simple"}
2025-07-27T15:03:42+02:00       INFO    Starting Controller     {"controller": "simple", "controllerGroup": "demo.demo.local", "controllerKind": "Simple"}
2025-07-27T15:03:42+02:00       INFO    Starting workers        {"controller": "simple", "controllerGroup": "demo.demo.local", "controllerKind": "Simple", "worker count": 1}
```

### ✅ Verify

- Terminal

```bash
❯ curl localhost:2112/metrics | grep reconcile

  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  8603    0  8603    0     0   9.8M      0 --:--:-- --:--:-- --:--:-- 8401k
# HELP reconcile_count Number of reconciliations performed
# TYPE reconcile_count counter
reconcile_count{cluster="minikube",namespace="metrics-ex"} 3
# HELP reconcile_errors Number reconciliation errors
# TYPE reconcile_errors counter
reconcile_errors{cluster="minikube",namespace="metrics-ex"} 1
```

- Or in the browser:
  - Open: <http://localhost:2112/metrics>
