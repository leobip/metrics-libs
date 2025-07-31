// metrics-common.go
package libs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Variables para almacenar los valores en memoria (visibles desde Prometheus y Kafka)
var (
	kubeClient                  *kubernetes.Clientset
	mu                          sync.Mutex
	reconcCount                 int
	reconcErrors                int
	lastSuccessfulReconcileTime time.Time
	healthStatus                float64 = 1 // 1 = healthy, 0 = unhealthy
)

type Tag struct {
	Name  string
	Value any
}

type Metric struct {
	Name  string
	Value any
	Type  string // "gauge", "counter", etc.
	/*
		 	Labels map[string]string
			Error  error
	*/
}

// GetKubeClient returns a cached Kubernetes clientset
func GetKubeClient() (*kubernetes.Clientset, error) {
	if kubeClient != nil {
		return kubeClient, nil
	}

	config, err := BuildKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	kubeClient = clientset
	return kubeClient, nil
}

// BuildKubeConfig handles in-cluster and out-of-cluster configs
func BuildKubeConfig() (*rest.Config, error) {
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		// running in cluster
		return rest.InClusterConfig()
	}
	// fallback: out-of-cluster (dev mode)
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func getMapValues() []Tag {
	return []Tag{
		{"namespace", os.Getenv("POD_NAMESPACE")},
		{"cluster", os.Getenv("CLUSTER_NAME")},
		{"resource_kind", os.Getenv("RESOURCE_KIND")},
		{"resource_name", os.Getenv("RESOURCE_NAME")},
		{"controller", os.Getenv("CONTROLLER_NAME")},
		{"controller_version", os.Getenv("CONTROLLER_VERSION")},
	}
}

func getMetrics() []Metric {
	return []Metric{
		{"memory_usage_bytes", getMemoryUsageBytes(), "gauge"},
		{"cpu_usage_cores", getCPUUsageCores(), "gauge"},
		{"health", getHealthStatus(), "gauge"},
		{"reconcile_count", getReconcileCount(), "counter"},
		{"reconcile_errors", getReconcileErrors(), "counter"},
		{"last_successful_reconcile_time", getLastSuccessfulReconcileTime(), "gauge"},
		// ...
	}
}

// Extrae el namespace del entorno o pone 'default'
func getNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	return "default"
}

var namespace = getNamespace()

// Extrae el cluster desde KUBECONTEXT o usa 'local'
func getClusterFromContext() string {
	if ctx := os.Getenv("KUBECONTEXT"); ctx != "" {
		return ctx
	}
	return "local"
}

func SetHealthStatus(healthy bool) {
	mu.Lock()
	defer mu.Unlock()
	if healthy {
		healthStatus = 1
	} else {
		healthStatus = 0
	}
}

func getHealthStatus() float64 {
	mu.Lock()
	defer mu.Unlock()
	return healthStatus
}

func getMemoryUsageBytes() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) // Memoria actualmente asignada
}

func getCPUUsageCores() float64 {
	return float64(runtime.NumGoroutine())
}

func getReconcileCount() float64 {
	mu.Lock()
	defer mu.Unlock()
	return float64(reconcCount)
}

func getReconcileErrors() float64 {
	mu.Lock()
	defer mu.Unlock()
	return float64(reconcErrors)
}

func getLastSuccessfulReconcileTime() float64 {
	mu.Lock()
	defer mu.Unlock()
	if lastSuccessfulReconcileTime.IsZero() {
		return 0
	}
	return float64(lastSuccessfulReconcileTime.Unix())
}

// Estas funciones las llamará el reconciler (desde el operador) cuando ocurra algo:
func IncReconcileCount() {
	mu.Lock()
	defer mu.Unlock()
	reconcCount++
	lastSuccessfulReconcileTime = time.Now()
}

func IncReconcileErrors() {
	mu.Lock()
	defer mu.Unlock()
	reconcErrors++
}

// Retorna un map con el número de pods por tipo de owner (Deployment, DaemonSet, etc.)
func GetPodCountsByController(clientset *kubernetes.Clientset, namespace string) (map[string]int, error) {

	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing pods: %w", err)
	}

	result := make(map[string]int)
	for _, pod := range pods.Items {
		ownerType := "Unknown"
		if len(pod.OwnerReferences) > 0 {
			ownerType = pod.OwnerReferences[0].Kind
		}
		result[ownerType]++
	}

	return result, nil
}
