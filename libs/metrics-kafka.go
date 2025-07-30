// metrics-kafka.go
package libs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	kafkaTopic   = os.Getenv("KAFKA_TOPIC")
	kafkaBroker  = os.Getenv("KAFKA_BROKER") // e.g., "kafka-service:9092"
	kafkaWriter  *kafka.Writer
	publishEvery = 5 * time.Minute
)

func StartKafkaMetrics() error {
	if kafkaTopic == "" || kafkaBroker == "" {
		return fmt.Errorf("missing KAFKA_TOPIC or KAFKA_BROKER environment variable")
	}
	fmt.Printf("🔧 Initializing Kafka writer with broker: %s and topic: %s\n", kafkaBroker, kafkaTopic)

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    kafkaTopic,
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
	// For testing, we can use hardcoded values or build a message dynamically
	// Uncomment the next line to use hardcoded values for testing
	msg := buildHardcodedMetricsMessage()
	// Or use the dynamic metrics collection
	// msg := buildMetricsMessage()
	fmt.Printf("🛠 Built message: %+v\n", msg)
	return sendToKafka(msg)
}

func buildHardcodedMetricsMessage() map[string]interface{} {
	fields := map[string]interface{}{
		"memory_usage_bytes":             123456789,
		"cpu_usage_cores":                0.85,
		"reconcile_count":                42,
		"reconcile_errors":               3,
		"last_successful_reconcile_time": float64(time.Now().Add(-1 * time.Minute).UnixMilli()),
	}
	tags := map[string]interface{}{
		"namespace":          "metrics-ex",
		"cluster":            "minikube",
		"resource_kind":      "Pod",
		"resource_name":      "example-pod",
		"controller":         "simple-operator",
		"controller_version": "v0.1.0",
	}
	message := map[string]interface{}{
		"name":      "pod_status",
		"timestamp": time.Now().UnixMilli(),
		"tags":      tags,
		"fields":    fields,
	}
	return message
}

func buildMetricsMessage() map[string]interface{} {
	metrics := getMetrics()
	mapValues := getMapValues()

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

	message := map[string]interface{}{
		"name":      "pod_status",
		"timestamp": time.Now().UnixMilli(),
		"tags":      tags,
		"fields":    fields,
	}

	return message
}

func sendToKafka(msg map[string]interface{}) error {
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
