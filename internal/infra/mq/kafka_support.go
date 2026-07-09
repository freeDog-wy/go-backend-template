package mq

import (
	"strings"

	"github.com/segmentio/kafka-go"
)

func normalizeKafkaBrokers(brokers []string) []string {
	normalizedBrokers := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		if trimmed := strings.TrimSpace(broker); trimmed != "" {
			normalizedBrokers = append(normalizedBrokers, trimmed)
		}
	}
	return normalizedBrokers
}

func newKafkaWriter(brokers []string, topic, clientID string) *kafka.Writer {
	normalizedBrokers := normalizeKafkaBrokers(brokers)
	if len(normalizedBrokers) == 0 {
		panic("kafka brokers must not be empty")
	}
	if strings.TrimSpace(topic) == "" {
		panic("kafka topic must not be empty")
	}

	return &kafka.Writer{
		Addr:         kafka.TCP(normalizedBrokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		Transport: &kafka.Transport{
			ClientID: strings.TrimSpace(clientID),
		},
	}
}
