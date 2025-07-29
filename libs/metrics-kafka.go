// homecalling-kafka.go
package homecalling

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

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    kafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}

	go func() {
		ticker := time.NewTicker(publishEvery)
		defer ticker.Stop()
		for {
			if err := sendMetricsToKafka(); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️ error sending metrics to Kafka: %v\n", err)
			}
			<-ticker.C
		}
	}()

	return nil
}

func sendMetricsToKafka() error {
	msg := buildMetricsMessage()
	return sendToKafka(msg)
}

func buildMetricsMessage() map[string]interface{} {
	metrics := getMetrics()

	fields := make(map[string]interface{})
	for _, m := range metrics {
		fields[m.Name] = m.Value
	}

	message := map[string]interface{}{
		"name":      "pod_status",
		"timestamp": time.Now().UnixMilli(),
		"tags": map[string]interface{}{
			"namespace":          os.Getenv("POD_NAMESPACE"),
			"cluster":            os.Getenv("CLUSTER_NAME"),
			"resource_kind":      os.Getenv("RESOURCE_KIND"),
			"resource_name":      os.Getenv("RESOURCE_NAME"),
			"controller":         os.Getenv("CONTROLLER_NAME"),
			"controller_version": os.Getenv("CONTROLLER_VERSION"),
		},
		"fields": fields,
	}

	return message
}

func sendToKafka(msg map[string]interface{}) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka message: %w", err)
	}

	kafkaMsg := kafka.Message{
		Key:   []byte("metrics"),
		Value: payload,
		Time:  time.Now(),
	}

	return kafkaWriter.WriteMessages(context.Background(), kafkaMsg)
}

/* func BuildMetricsMessage() map[string]interface{} {
	timestamp := time.Now().UnixMilli()

	tags := make(map[string]interface{})
	fields := make(map[string]interface{})

	// Reunimos métricas de pod
	if podMetrics, err := getPodStatusMetrics(); err == nil {
		tags["namespace"] = podMetrics.Namespace
		tags["pod"] = podMetrics.PodName
		tags["host_ip"] = podMetrics.HostIP
		tags["image"] = podMetrics.Image
		tags["phase"] = podMetrics.Phase
		tags["container"] = podMetrics.ContainerName

		fields["cpu_milli"] = podMetrics.CPUMilli
		fields["mem_mib"] = podMetrics.MemoryMiB
		fields["ready"] = podMetrics.Ready
		fields["restart_count"] = podMetrics.RestartCount
		fields["started"] = podMetrics.Started
	} else {
		fmt.Fprintf(os.Stderr, "⚠️ error getting pod metrics: %v\n", err)
	}

	return map[string]interface{}{
		"name":      "pod_status",
		"timestamp": timestamp,
		"tags":      tags,
		"fields":    fields,
	}
}

func SendToKafka(message map[string]interface{}) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling message: %w", err)
	}

	kafkaMsg := kafka.Message{
		Value: payload,
	}

	return kafkaProducer.WriteMessages(context.Background(), kafkaMsg)
}
*/
