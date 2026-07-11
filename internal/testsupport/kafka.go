package testsupport

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/pkg/envfile"
	kgo "github.com/segmentio/kafka-go"
)

const kafkaBrokersEnv = "TEST_KAFKA_BROKERS"

// Kafka provides isolated topic lifecycle helpers for Kafka integration tests.
type Kafka struct {
	Brokers []string
}

// OpenKafka verifies the explicitly configured Kafka broker connection.
func OpenKafka(t testing.TB) *Kafka {
	t.Helper()
	if err := envfile.LoadNearest(".env"); err != nil {
		t.Fatalf("load nearest .env: %v", err)
	}

	brokers := splitBrokers(os.Getenv(kafkaBrokersEnv))
	if len(brokers) == 0 {
		t.Fatalf("%s must be set for Kafka integration tests", kafkaBrokersEnv)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := kgo.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		t.Fatalf("connect to Kafka broker %s: %v", brokers[0], err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close Kafka broker connection: %v", err)
	}
	return &Kafka{Brokers: brokers}
}

// CreateTopic creates a one-partition topic unique to the current test and
// schedules best-effort deletion after the test completes.
func (k *Kafka) CreateTopic(t testing.TB, prefix string) string {
	t.Helper()
	topic := fmt.Sprintf("%s.%d", strings.Trim(prefix, "."), time.Now().UnixNano())
	conn := k.dial(t)
	err := conn.CreateTopics(kgo.TopicConfig{Topic: topic, NumPartitions: 1, ReplicationFactor: 1})
	closeErr := conn.Close()
	if err != nil {
		t.Fatalf("create Kafka topic %s: %v", topic, err)
	}
	if closeErr != nil {
		t.Fatalf("close Kafka topic connection: %v", closeErr)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, err := kgo.DialContext(ctx, "tcp", k.Brokers[0])
		if err != nil {
			t.Logf("connect to delete Kafka topic %s: %v", topic, err)
			return
		}
		defer conn.Close()
		if err := conn.DeleteTopics(topic); err != nil {
			t.Logf("delete Kafka topic %s: %v", topic, err)
		}
	})
	return topic
}

func (k *Kafka) dial(t testing.TB) *kgo.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := kgo.DialContext(ctx, "tcp", k.Brokers[0])
	if err != nil {
		t.Fatalf("connect to Kafka broker %s: %v", k.Brokers[0], err)
	}
	return conn
}

func splitBrokers(value string) []string {
	parts := strings.Split(value, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		if broker := strings.TrimSpace(part); broker != "" {
			brokers = append(brokers, broker)
		}
	}
	return brokers
}
