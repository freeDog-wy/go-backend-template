package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/config"
	domainAudit "github.com/freeDog-wy/go-backend-template/internal/domain/audit"
	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"
	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
	"github.com/freeDog-wy/go-backend-template/internal/infra/cache"
	"github.com/freeDog-wy/go-backend-template/internal/infra/database"
	"github.com/freeDog-wy/go-backend-template/internal/infra/logging"
	"github.com/freeDog-wy/go-backend-template/internal/infra/mq"
	RepoAudit "github.com/freeDog-wy/go-backend-template/internal/repository/audit"
	RepoConsumption "github.com/freeDog-wy/go-backend-template/internal/repository/consumption"
	SvcAudit "github.com/freeDog-wy/go-backend-template/internal/usecase/audit"
	SvcVerification "github.com/freeDog-wy/go-backend-template/internal/usecase/verification"
	"github.com/freeDog-wy/go-backend-template/pkg/email"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
	"gorm.io/gorm"
)

// Worker 事件消费者进程。
type Worker struct {
	consumer mq.Consumer
}

// Run 启动消费者，阻塞直到 ctx 取消。
func (w *Worker) Run(ctx context.Context) error {
	return w.consumer.Run(ctx)
}

func initWorker(cfg *config.Config) *Worker {
	appLogger := logging.Init(cfg.App.Mode)
	db := database.NewPostgresDB(cfg.Database.DSN)

	emailSender := email.New(email.Config{
		SmtpHost:     cfg.Email.SmtpHost,
		SmtpPort:     cfg.Email.SmtpPort,
		SmtpUser:     cfg.Email.SmtpUser,
		SmtpPassword: cfg.Email.SmtpPassword,
		FromAddress:  cfg.Email.FromAddress,
	})

	verificationConsumer := SvcVerification.NewConsumer(emailSender, cfg.Email.SiteBaseURL, appLogger)
	auditConsumer := SvcAudit.NewConsumer(RepoAudit.New(db), appLogger)

	consumer := initWorkerMQConsumer(cfg, db, appLogger)

	consumer.Handle("user.registered", func(ctx context.Context, message mq.Message) error {
		var evt domainIdentity.Registered
		if err := json.Unmarshal(message.Payload, &evt); err != nil {
			return err
		}
		return nil
	})

	consumer.Handle("user.email_verification_requested", func(ctx context.Context, message mq.Message) error {
		var evt domainVerification.EmailVerificationRequested
		if err := json.Unmarshal(message.Payload, &evt); err != nil {
			return err
		}
		return verificationConsumer.OnEmailVerificationRequested(ctx, evt)
	})

	consumer.Handle("user.password_reset_requested", func(ctx context.Context, message mq.Message) error {
		var evt domainVerification.PasswordResetRequested
		if err := json.Unmarshal(message.Payload, &evt); err != nil {
			return err
		}
		return verificationConsumer.OnPasswordResetRequested(ctx, evt)
	})

	consumer.Handle("audit.log.requested", func(ctx context.Context, message mq.Message) error {
		var evt domainAudit.LogRequested
		if err := json.Unmarshal(message.Payload, &evt); err != nil {
			return err
		}
		return auditConsumer.OnLogRequested(ctx, evt)
	})

	return &Worker{consumer: consumer}
}

func initWorkerMQConsumer(cfg *config.Config, db *gorm.DB, appLogger logger.Logger) mq.Consumer {
	switch strings.ToLower(strings.TrimSpace(cfg.MQ.Provider)) {
	case "", "redis":
		rdb, err := cache.NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			panic("failed to init redis: " + err.Error())
		}
		return mq.NewRedisConsumer(rdb, cfg.MQ.EventsName, cfg.Worker.ConsumerGroup, cfg.Worker.ConsumerName, appLogger, mq.ConsumerConfig{
			ReadCount:           int64(cfg.Worker.ConsumerReadCount),
			ReadBlock:           time.Duration(cfg.Worker.ConsumerReadBlockSeconds) * time.Second,
			PendingMinIdle:      time.Duration(cfg.Worker.ConsumerPendingMinIdleSeconds) * time.Second,
			PendingReclaimBatch: int64(cfg.Worker.ConsumerReclaimBatch),
			MaxRetries:          int64(cfg.Worker.ConsumerMaxRetries),
			IdempotencyTTL:      time.Duration(cfg.Worker.ConsumerIdempotencyTTLHours) * time.Hour,
			ProcessingLockTTL:   time.Duration(cfg.Worker.ConsumerProcessingLockSeconds) * time.Second,
			DeadLetterStream:    cfg.Worker.DeadLetterStream,
		})
	case "kafka":
		return mq.NewKafkaConsumer(
			cfg.MQ.Kafka.Brokers,
			cfg.MQ.EventsName,
			appLogger,
			RepoConsumption.New(db),
			mq.KafkaConsumerConfig{
				GroupID:           cfg.Worker.ConsumerGroup,
				ClientID:          cfg.MQ.Kafka.ClientID,
				MinBytes:          cfg.Worker.KafkaReadMinBytes,
				MaxBytes:          cfg.Worker.KafkaReadMaxBytes,
				MaxWait:           time.Duration(cfg.Worker.KafkaMaxWaitSeconds) * time.Second,
				ProcessingLockTTL: time.Duration(cfg.Worker.ConsumerProcessingLockSeconds) * time.Second,
				MaxRetries:        cfg.Worker.ConsumerMaxRetries,
				RetryTopic:        cfg.Worker.KafkaRetryTopic,
				DeadLetterTopic:   cfg.Worker.KafkaDeadLetterTopic,
			},
		)
	default:
		panic("unsupported mq provider: " + cfg.MQ.Provider)
	}
}
