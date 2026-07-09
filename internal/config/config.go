package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App            AppConfig
	Server         ServerConfig
	Database       DatabaseConfig
	MQ             MQConfig
	Redis          RedisConfig
	Worker         WorkerConfig
	Auth           AuthConfig
	Email          EmailConfig
	Captcha        CaptchaConfig
	Cron           CronConfig
	Tracing        TracingConfig
	BootstrapAdmin BootstrapAdminConfig `mapstructure:"bootstrap_admin"`
}

type AuthConfig struct {
	JWTIssuer             string
	JWTAudience           string
	JWTSecret             string
	AccessTokenTTLMinutes int
	RefreshTokenTTLHours  int
	LoginFailThreshold    int
}

type DatabaseConfig struct {
	DSN string
}

type MQConfig struct {
	Provider   string
	EventsName string      `mapstructure:"events_name"`
	Kafka      KafkaConfig `mapstructure:"kafka"`
}

type KafkaConfig struct {
	Brokers  []string
	ClientID string `mapstructure:"client_id"`
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type AppConfig struct {
	Mode string
}

type WorkerConfig struct {
	ConsumerGroup                 string `mapstructure:"consumer_group"`
	ConsumerName                  string `mapstructure:"consumer_name"`
	ConsumerReadCount             int    `mapstructure:"consumer_read_count"`
	ConsumerReadBlockSeconds      int    `mapstructure:"consumer_read_block_seconds"`
	ConsumerPendingMinIdleSeconds int    `mapstructure:"consumer_pending_min_idle_seconds"`
	ConsumerReclaimBatch          int    `mapstructure:"consumer_reclaim_batch"`
	ConsumerMaxRetries            int    `mapstructure:"consumer_max_retries"`
	ConsumerIdempotencyTTLHours   int    `mapstructure:"consumer_idempotency_ttl_hours"`
	ConsumerProcessingLockSeconds int    `mapstructure:"consumer_processing_lock_seconds"`
	KafkaReadMinBytes             int    `mapstructure:"kafka_read_min_bytes"`
	KafkaReadMaxBytes             int    `mapstructure:"kafka_read_max_bytes"`
	KafkaMaxWaitSeconds           int    `mapstructure:"kafka_max_wait_seconds"`
	KafkaRetryTopic               string `mapstructure:"kafka_retry_topic"`
	KafkaDeadLetterTopic          string `mapstructure:"kafka_dead_letter_topic"`
	DeadLetterStream              string `mapstructure:"dead_letter_stream"`
}

type ServerConfig struct {
	IP             string
	Port           int
	ReadTimeout    int
	WriteTimeout   int
	TrustedProxies []string
}

type EmailConfig struct {
	SmtpHost     string
	SmtpPort     int
	SmtpUser     string
	SmtpPassword string
	FromAddress  string
	SiteBaseURL  string
}

type CaptchaConfig struct {
	Width  int
	Height int
	Length int
}

type CronConfig struct {
	Enabled                            bool
	OutboxPublishIntervalSeconds       int `mapstructure:"outbox_publish_interval_seconds"`
	OutboxBatchSize                    int `mapstructure:"outbox_batch_size"`
	VerificationCleanupIntervalSeconds int `mapstructure:"verification_cleanup_interval_seconds"`
}

type TracingConfig struct {
	Endpoint string // Jaeger OTLP HTTP 地址，为空时回退到 stdout
}

type BootstrapAdminConfig struct {
	Enabled  bool
	Name     string
	Email    string
	Password string
}

func Load(configPath string) *Config {
	v := viper.New()

	// set default config
	v.SetDefault("app.mode", "development")
	v.SetDefault("server.ip", "localhost")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.readTimeout", 30)
	v.SetDefault("server.writeTimeout", 30)
	v.SetDefault("email.smtpPort", 465)
	v.SetDefault("email.siteBaseURL", "http://localhost:5173")
	v.SetDefault("mq.provider", "redis")
	v.SetDefault("mq.events_name", "domain.events")
	v.SetDefault("mq.kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("mq.kafka.client_id", "go-backend-template")
	v.SetDefault("auth.jwtIssuer", "go-backend-template")
	v.SetDefault("auth.jwtAudience", "go-backend-template-client")
	v.SetDefault("auth.jwtSecret", "change-me")
	v.SetDefault("auth.accessTokenTTLMinutes", 15)
	v.SetDefault("auth.refreshTokenTTLHours", 24*7)
	v.SetDefault("auth.loginFailThreshold", 5)
	v.SetDefault("worker.consumer_group", "user-worker")
	v.SetDefault("worker.consumer_name", "worker-1")
	v.SetDefault("worker.consumer_read_count", 10)
	v.SetDefault("worker.consumer_read_block_seconds", 1)
	v.SetDefault("worker.consumer_pending_min_idle_seconds", 30)
	v.SetDefault("worker.consumer_reclaim_batch", 10)
	v.SetDefault("worker.consumer_max_retries", 10)
	v.SetDefault("worker.consumer_idempotency_ttl_hours", 24*7)
	v.SetDefault("worker.consumer_processing_lock_seconds", 300)
	v.SetDefault("worker.kafka_read_min_bytes", 1024)
	v.SetDefault("worker.kafka_read_max_bytes", 10*1024*1024)
	v.SetDefault("worker.kafka_max_wait_seconds", 1)
	v.SetDefault("worker.kafka_retry_topic", "domain.events.retry")
	v.SetDefault("worker.kafka_dead_letter_topic", "domain.events.dlq")
	v.SetDefault("worker.dead_letter_stream", "domain.events.dlq")

	v.SetDefault("captcha.width", 120)
	v.SetDefault("captcha.height", 40)
	v.SetDefault("captcha.length", 6)
	v.SetDefault("cron.enabled", true)
	v.SetDefault("cron.outbox_publish_interval_seconds", 5)
	v.SetDefault("cron.outbox_batch_size", 100)
	v.SetDefault("cron.verification_cleanup_interval_seconds", 300)
	v.SetDefault("bootstrap_admin.enabled", false)
	v.SetDefault("bootstrap_admin.name", "Admin")
	v.SetDefault("bootstrap_admin.email", "")
	v.SetDefault("bootstrap_admin.password", "12345678")

	// load config file
	if configPath == "" {
		configPath = "config.yaml"
	}

	// If configPath contains a path separator, treat it as a direct file path.
	// Otherwise, search for it in known config directories.
	if strings.Contains(configPath, string(os.PathSeparator)) {
		v.SetConfigFile(configPath)
	} else {
		name := strings.TrimSuffix(configPath, ".yaml")
		v.SetConfigName(name)
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./internal/config")
		v.AddConfigPath("../internal/config")
	}

	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("failed to read config file (%s): %v", configPath, err))
	}

	// load Env
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %v", err))
	}

	return &cfg
}
