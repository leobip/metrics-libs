// local-test/main.go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/leobip/metrics-libs/libs"
)

func main() {
	fmt.Println("🔄 Starting Kafka metrics test...")

	if err := libs.StartKafkaMetrics(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to start Kafka metrics: %v\n", err)
		return
	}

	// Wait to let at least one message be sent
	fmt.Println("⏳ Waiting 10 seconds to send message...")
	time.Sleep(10 * time.Second)
	fmt.Println("✅ Message should be visible in Kafka-UI topic: metrics")
}
