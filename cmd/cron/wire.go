package main

import (
	"context"
	"strings"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/config"
	"github.com/freeDog-wy/go-backend-template/internal/infra/cache"
	"github.com/freeDog-wy/go-backend-template/internal/infra/database"
	"github.com/freeDog-wy/go-backend-template/internal/infra/logging"
	"github.com/freeDog-wy/go-backend-template/internal/infra/mq"
	RepoOutbox "github.com/freeDog-wy/go-backend-template/internal/repository/outbox"
	RepoVerification "github.com/freeDog-wy/go-backend-template/internal/repository/verification"
	UsecaseSupport "github.com/freeDog-wy/go-backend-template/internal/usecase/support"
	UsecaseVerification "github.com/freeDog-wy/go-backend-template/internal/usecase/verification"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
	"github.com/freeDog-wy/go-backend-template/pkg/scheduler"
)

type CronApp struct {
	enabled bool
	logger  logger.Logger
	runner  *scheduler.Runner
}

func (a *CronApp) Run(ctx context.Context) error {
	if !a.enabled {
		a.logger.Info("cron is disabled by configuration")
		<-ctx.Done()
		return ctx.Err()
	}

	return a.runner.Run(ctx)
}

func initCronApp(cfg *config.Config) *CronApp {
	appLogger := logging.Init(cfg.App.Mode)
	runner := scheduler.New(appLogger)
	if cfg.Cron.Enabled {
		if cfg.Cron.OutboxPublishIntervalSeconds <= 0 {
			panic("cron.outbox_publish_interval_seconds must be greater than zero")
		}
		if cfg.Cron.OutboxBatchSize <= 0 {
			panic("cron.outbox_batch_size must be greater than zero")
		}
		if cfg.Cron.VerificationCleanupIntervalSeconds <= 0 {
			panic("cron.verification_cleanup_interval_seconds must be greater than zero")
		}

		db := database.NewPostgresDB(cfg.Database.DSN)

		outboxRepo := RepoOutbox.New(db)
		publisher := initOutboxMQPublisher(cfg, appLogger)
		outboxPublisher := UsecaseSupport.NewOutboxPublisher(
			outboxRepo,
			mq.NewOutboxPublisherAdapter(publisher),
			appLogger,
			cfg.Cron.OutboxBatchSize,
		)
		verificationRepo := RepoVerification.New(db)
		verificationCron := UsecaseVerification.NewCron(verificationRepo, appLogger)

		if err := runner.Register(scheduler.Job{
			Name:     "outbox.publish_pending_events",
			Interval: time.Duration(cfg.Cron.OutboxPublishIntervalSeconds) * time.Second,
			Run:      outboxPublisher.PublishPending,
		}); err != nil {
			panic("failed to register outbox publisher job: " + err.Error())
		}

		if err := runner.Register(scheduler.Job{
			Name:     "verification.cleanup_expired_tokens",
			Interval: time.Duration(cfg.Cron.VerificationCleanupIntervalSeconds) * time.Second,
			Run:      verificationCron.CleanupExpiredTokens,
		}); err != nil {
			panic("failed to register verification cleanup job: " + err.Error())
		}
	}

	return &CronApp{
		enabled: cfg.Cron.Enabled,
		logger:  appLogger,
		runner:  runner,
	}
}

func initOutboxMQPublisher(cfg *config.Config, appLogger logger.Logger) mq.Publisher {
	switch strings.ToLower(strings.TrimSpace(cfg.MQ.Provider)) {
	case "", "redis":
		rdb, err := cache.NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			panic("failed to init redis: " + err.Error())
		}
		return mq.NewRedisPublisher(rdb, cfg.MQ.EventsName, appLogger)
	case "kafka":
		return mq.NewKafkaPublisher(cfg.MQ.Kafka.Brokers, cfg.MQ.EventsName, cfg.MQ.Kafka.ClientID, appLogger)
	default:
		panic("unsupported mq provider: " + cfg.MQ.Provider)
	}
}
