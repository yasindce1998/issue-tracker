// Package messaging provides factories and utilities for creating message broker instances.
// It handles the selection and initialization of appropriate messaging implementations.
package messaging

import (
	"os"
	"strings"

	"github.com/yasindce1998/issue-tracker/pkg/messaging/broker"
	"github.com/yasindce1998/issue-tracker/pkg/messaging/kfkimp"
	"github.com/yasindce1998/issue-tracker/pkg/messaging/memory"
)

// NewMessageBroker creates a message broker based on environment configuration
func NewMessageBroker() (broker.MessageBroker, error) {
	communicationMethod := os.Getenv("COMMUNICATION_METHOD")

	switch strings.ToLower(communicationMethod) {
	case "kafka":
		// Get Kafka configuration from environment
		kafkaBrokers := os.Getenv("KAFKA_BROKERS")
		if kafkaBrokers == "" {
			kafkaBrokers = "localhost:9092" // Default
		}

		topicPrefix := os.Getenv("KAFKA_TOPIC_PREFIX")
		if topicPrefix == "" {
			topicPrefix = "issue-tracker" // Default
		}

		return kfkimp.NewKafkaBroker(strings.Split(kafkaBrokers, ","), topicPrefix)
	default: // "stream" or empty
		return memory.NewInMemoryBroker(), nil
	}
}
