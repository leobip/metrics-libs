// homecalling-common.go
package homecalling

import (
	"context"
	"fmt"
	"log"
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

type Config struct {
	clusterName                 string
	nameSpace                   string
	resourceKind                string
	controllerName              string
	controllerVersion           string
	kafkaBroker                 string
	kafkaTopic                  string
	reconcCount                 int
	reconcErrors                int
	lastSuccessfulReconcileTime time.Time
	healthStatus                float64

	kubeClient *kubernetes.Clientset
	mu         sync.Mutex
}

var cfg Config

func InitLibrary() {
	var err error

	cfg.clusterName = getEnv("METRICS_CLUSTER", "local")
	cfg.nameSpace = getEnv("METRICS_NAMESPACE", "default")
	cfg.resourceKind = getEnv("RESOURCE_KIND", "unknown")
	cfg.controllerName = getEnv("CONTROLLER_NAME", "unknown")
	cfg.controllerVersion = getEnv("CONTROLLER_VERSION", "v1.0.0")
	cfg.kafkaBroker = getEnv("KAFKA_BROKER", "kafka-service:9092")
	cfg.kafkaTopic = getEnv("KAFKA_TOPIC", "homecalling-metrics")
	cfg.healthStatus = 1 // ✅ default saludable

	cfg.kubeClient, err = GetKubeClient()
	if err != nil {
		log.Fatalf("❌ Failed to get Kubernetes client: %v", err)
	}

	log.Printf("🔧 Config initialized: cluster=%s, namespace=%s, resourceKind=%s, controller=%s, kafkaBroker=%s, kafkaTopic=%s",
		cfg.clusterName, cfg.nameSpace, cfg.resourceKind, cfg.controllerName, cfg.kafkaBroker, cfg.kafkaTopic)

	if err := InitKafka(); err != nil {
		log.Fatalf("❌ Failed to initialize Kafka: %v", err)
	}
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func InitKafka() error {
	if cfg.kafkaTopic == "" || cfg.kafkaBroker == "" {
		return fmt.Errorf("missing KAFKA_TOPIC or KAFKA_BROKER environment variable")
	}
	log.Printf("🔧 Initializing Kafka writer with broker: %s and topic: %s", cfg.kafkaBroker, cfg.kafkaTopic)
	return StartKafkaMetrics()
}

type Tag struct {
	Name  string
	Value any
}

type Metric struct {
	Name  string
	Value any
	Type  string // "gauge", "counter", etc.
}

// GetKubeClient returns a cached Kubernetes clientset
func GetKubeClient() (*kubernetes.Clientset, error) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	if cfg.kubeClient != nil {
		return cfg.kubeClient, nil
	}

	config, err := BuildKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	cfg.kubeClient = clientset
	return cfg.kubeClient, nil
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
		{"namespace", cfg.nameSpace},
		{"cluster", cfg.clusterName},
		{"resource_kind", cfg.resourceKind},
		//{"resource_name", cfg.resourceName}, // Uncomment if needed
		{"controller", cfg.controllerName},
		{"controller_version", cfg.controllerVersion},
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

func SetHealthStatus(healthy bool) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	if healthy {
		cfg.healthStatus = 1
	} else {
		cfg.healthStatus = 0
	}
}

func getHealthStatus() float64 {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.healthStatus
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
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return float64(cfg.reconcCount)
}

func getReconcileErrors() float64 {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return float64(cfg.reconcErrors)
}

func getLastSuccessfulReconcileTime() float64 {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	if cfg.lastSuccessfulReconcileTime.IsZero() {
		return 0
	}
	return float64(cfg.lastSuccessfulReconcileTime.Unix())
}

// Estas funciones las llamará el reconciler (desde el operador) cuando ocurra algo:
func IncReconcileCount() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.reconcCount++
	cfg.lastSuccessfulReconcileTime = time.Now()
}

func IncReconcileErrors() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.reconcErrors++
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


// homecalling-kafka.go
package homecalling

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

var ( // e.g., "kafka-service:9092"
	kafkaWriter  *kafka.Writer
	publishEvery = 5 * time.Minute
)

func StartKafkaMetrics() error {
	if cfg.kafkaTopic == "" || cfg.kafkaBroker == "" {
		return fmt.Errorf("missing KAFKA_TOPIC or KAFKA_BROKER environment variable")
	}
	fmt.Printf("🔧 Initializing Kafka writer with broker: %s and topic: %s\n", cfg.kafkaBroker, cfg.kafkaTopic)

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(cfg.kafkaBroker),
		Topic:    cfg.kafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}

	go func() {
		ticker := time.NewTicker(publishEvery)
		defer ticker.Stop()
		for {
			fmt.Println("📤 Sending metrics to Kafka...")
			if err := sendMetricsToKafka(); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️ error sending metrics to Kafka: %v\n", err)
			} else {
				fmt.Println("✅ Metrics sent successfully to Kafka")
			}
			<-ticker.C
		}
	}()

	return nil
}

func sendMetricsToKafka() error {
	msg := buildMetricsMessage()
	fmt.Printf("🛠 Built message: %+v\n", msg)
	return sendToKafka(msg)
}

func buildMetricsMessage() map[string]interface{} {
	metrics := getMetrics()
	mapValues := getMapValues()

	var podCounts map[string]int
	if cfg.kubeClient == nil {
		log.Println("⚠️ Kubernetes client is not initialized, skipping pod counts")
	} else {
		var err error
		// Get pod counts by controller type
		log.Println("🔍 Fetching pod counts by controller type...")
		// If cfg.kubeClient is not nil, we proceed to get pod counts.
		log.Printf("Using Kubernetes client: %v", cfg.kubeClient)
		podCounts, err = GetPodCountsByController(cfg.kubeClient, cfg.nameSpace)
		if err != nil {
			log.Printf("⚠️ Could not get pod counts: %v", err)
		}
	}

	fmt.Printf("📊 Collected %d tags\n", len(mapValues))
	fmt.Printf("📊 Collected %d metrics\n", len(metrics))

	tags := make(map[string]interface{})
	for _, tag := range mapValues {
		tags[tag.Name] = tag.Value
	}

	fields := make(map[string]interface{})
	for _, m := range metrics {
		fields[m.Name] = m.Value
	}

	if podCounts != nil {
		fields["Pods-Count"] = podCounts
	}

	message := map[string]interface{}{
		"name":      "pod_status",
		"timestamp": time.Now().UnixMilli(),
		"tags":      tags,
		"fields":    fields,
	}

	return message
}

func sendToKafka(msg map[string]any) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka message: %w", err)
	}

	fmt.Printf("📦 Final Kafka payload: %s\n", string(payload))

	kafkaMsg := kafka.Message{
		Key:   []byte("metrics"),
		Value: payload,
		Time:  time.Now(),
	}

	fmt.Println("📡 Writing message to Kafka...")
	err = kafkaWriter.WriteMessages(context.Background(), kafkaMsg)
	if err != nil {
		return fmt.Errorf("❌ failed to write message to Kafka: %w", err)
	}
	return nil
}



