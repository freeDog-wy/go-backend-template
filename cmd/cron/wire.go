package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/config"
	HdlHealth "github.com/freeDog-wy/go-backend-template/internal/handler/health"
	"github.com/freeDog-wy/go-backend-template/internal/infra/database"
	"github.com/freeDog-wy/go-backend-template/internal/infra/logging"
	"github.com/freeDog-wy/go-backend-template/internal/infra/mq"
	InfraStorage "github.com/freeDog-wy/go-backend-template/internal/infra/storage"
	"github.com/freeDog-wy/go-backend-template/internal/infra/tracing"
	RepoMedia "github.com/freeDog-wy/go-backend-template/internal/repository/media"
	RepoOutbox "github.com/freeDog-wy/go-backend-template/internal/repository/outbox"
	RepoVerification "github.com/freeDog-wy/go-backend-template/internal/repository/verification"
	UsecaseMedia "github.com/freeDog-wy/go-backend-template/internal/usecase/media"
	UsecaseMessaging "github.com/freeDog-wy/go-backend-template/internal/usecase/messaging"
	UsecaseSupport "github.com/freeDog-wy/go-backend-template/internal/usecase/support"
	UsecaseVerification "github.com/freeDog-wy/go-backend-template/internal/usecase/verification"
	pkgkafka "github.com/freeDog-wy/go-backend-template/pkg/kafka"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
	"github.com/freeDog-wy/go-backend-template/pkg/scheduler"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/gorm"
)

type CronApp struct {
	enabled     bool
	logger      logger.Logger
	runner      *scheduler.Runner
	probeServer *HdlHealth.Server
	running     atomic.Bool
	tp          *sdktrace.TracerProvider
}

func (a *CronApp) Run(ctx context.Context) error {
	a.running.Store(true)
	defer a.running.Store(false)
	if !a.enabled {
		a.logger.Info("cron is disabled by configuration")
		<-ctx.Done()
		return ctx.Err()
	}

	return a.runner.Run(ctx)
}

func (a *CronApp) ServeProbe() error {
	return a.probeServer.Serve()
}

func (a *CronApp) Shutdown(ctx context.Context) error {
	err := a.probeServer.Shutdown(ctx)
	tracing.Shutdown(ctx, a.tp)
	return err
}

func initCronApp(cfg *config.Config) (*CronApp, error) {
	tp, err := tracing.Init(cfg.App.Mode, cfg.Tracing.Endpoint, "go-backend-template-cron")
	if err != nil {
		return nil, fmt.Errorf("initialize tracing: %w", err)
	}

	appLogger := logging.Init(cfg.App.Mode)
	runner := scheduler.New(appLogger)
	cronApp := &CronApp{
		enabled: cfg.Cron.Enabled,
		logger:  appLogger,
		runner:  runner,
		tp:      tp,
	}
	checks := map[string]HdlHealth.Checker{
		"scheduler": HdlHealth.CheckFunc(func(context.Context) error {
			if !cronApp.running.Load() {
				return errors.New("scheduler loop is not running")
			}
			return nil
		}),
	}
	if cfg.Cron.Enabled {
		if cfg.Cron.OutboxPublishIntervalSeconds <= 0 {
			return nil, fmt.Errorf("cron.outbox_publish_interval_seconds must be greater than zero")
		}
		if cfg.Cron.OutboxBatchSize <= 0 {
			return nil, fmt.Errorf("cron.outbox_batch_size must be greater than zero")
		}
		if cfg.Cron.VerificationCleanupIntervalSeconds <= 0 {
			return nil, fmt.Errorf("cron.verification_cleanup_interval_seconds must be greater than zero")
		}

		db, err := database.NewPostgresDB(cfg.Database.DSN)
		if err != nil {
			return nil, fmt.Errorf("initialize postgres: %w", err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("get postgres health check handle: %w", err)
		}
		checks["database"] = HdlHealth.CheckFunc(sqlDB.PingContext)
		checks["kafka"] = HdlHealth.CheckFunc(func(ctx context.Context) error {
			return mq.PingKafka(ctx, cfg.MQ.Kafka.Brokers)
		})

		outboxRepo := RepoOutbox.New(db)
		publisher, err := mq.NewPublisher(mq.KafkaOptions{Brokers: cfg.MQ.Kafka.Brokers, Topic: cfg.MQ.EventsName, ClientID: cfg.MQ.Kafka.ClientID}, appLogger)
		if err != nil {
			return nil, fmt.Errorf("initialize kafka publisher: %w", err)
		}
		outboxPublisher := UsecaseSupport.NewOutboxPublisher(
			outboxRepo,
			mq.NewOutboxPublisherAdapter(publisher),
			appLogger,
			cfg.Cron.OutboxBatchSize,
		)
		verificationRepo := RepoVerification.New(db)
		verificationCron := UsecaseVerification.NewCron(verificationRepo, appLogger)
		if err := registerMediaCleanupJob(cfg, appLogger, runner, db); err != nil {
			return nil, err
		}

		if err := runner.Register(scheduler.Job{
			Name:     "outbox.publish_pending_events",
			Interval: time.Duration(cfg.Cron.OutboxPublishIntervalSeconds) * time.Second,
			Run:      outboxPublisher.PublishPending,
		}); err != nil {
			return nil, fmt.Errorf("register outbox publisher job: %w", err)
		}

		if err := runner.Register(scheduler.Job{
			Name:     "verification.cleanup_expired_tokens",
			Interval: time.Duration(cfg.Cron.VerificationCleanupIntervalSeconds) * time.Second,
			Run:      verificationCron.CleanupExpiredTokens,
		}); err != nil {
			return nil, fmt.Errorf("register verification cleanup job: %w", err)
		}

		if err := registerKafkaDLQJobs(cfg, appLogger, runner); err != nil {
			return nil, err
		}
	}

	cronApp.probeServer = HdlHealth.NewServer(cfg.Cron.Probe.Address(), checks, 2*time.Second)
	return cronApp, nil
}

func registerMediaCleanupJob(cfg *config.Config, appLogger logger.Logger, runner *scheduler.Runner, db *gorm.DB) error {
	storage, err := InfraStorage.NewS3(context.Background(), InfraStorage.Options{
		Endpoint:          cfg.Storage.S3.Endpoint,
		Region:            cfg.Storage.S3.Region,
		AccessKeyID:       cfg.Storage.S3.AccessKeyID,
		SecretAccessKey:   cfg.Storage.S3.SecretAccessKey,
		Bucket:            cfg.Storage.S3.Bucket,
		PublicBaseURL:     cfg.Storage.S3.PublicBaseURL,
		Prefix:            cfg.Storage.S3.Prefix,
		UsePathStyle:      cfg.Storage.S3.UsePathStyle,
		PresignTTLMinutes: cfg.Storage.S3.PresignTTLMinutes,
	})
	if errors.Is(err, InfraStorage.ErrNotConfigured) {
		appLogger.Info("media upload cleanup is disabled because S3 storage is not configured")
		return nil
	}
	if err != nil {
		return fmt.Errorf("initialize S3 storage for media cleanup: %w", err)
	}
	if cfg.Cron.MediaUploadCleanupIntervalSeconds <= 0 {
		return fmt.Errorf("cron.media_upload_cleanup_interval_seconds must be greater than zero")
	}
	if cfg.Cron.MediaUploadCleanupBatchSize <= 0 {
		return fmt.Errorf("cron.media_upload_cleanup_batch_size must be greater than zero")
	}
	service := UsecaseMedia.New(database.NewTxManager(db), RepoMedia.New(db), storage)
	if err := runner.Register(scheduler.Job{
		Name:     "media.cleanup_stale_uploads",
		Interval: time.Duration(cfg.Cron.MediaUploadCleanupIntervalSeconds) * time.Second,
		Run: func(ctx context.Context) error {
			_, err := service.CleanupStaleUploads(ctx, cfg.Cron.MediaUploadCleanupBatchSize)
			return err
		},
	}); err != nil {
		return fmt.Errorf("register media cleanup job: %w", err)
	}
	return nil
}

func registerKafkaDLQJobs(cfg *config.Config, appLogger logger.Logger, runner *scheduler.Runner) error {
	if cfg.Cron.DLQInspectionEnabled {
		if cfg.Cron.DLQInspectionIntervalSeconds <= 0 {
			return fmt.Errorf("cron.dlq_inspection_interval_seconds must be greater than zero")
		}
		if cfg.Cron.DLQInspectionBatchSize <= 0 {
			return fmt.Errorf("cron.dlq_inspection_batch_size must be greater than zero")
		}
		if strings.TrimSpace(cfg.Cron.DLQInspectionGroup) == "" {
			return fmt.Errorf("cron.dlq_inspection_group must not be empty")
		}

		inspector, err := mq.NewDeadLetterInspector(deadLetterOptions(cfg, cfg.Cron.DLQInspectionGroup), appLogger)
		if err != nil {
			return fmt.Errorf("initialize kafka dead letter inspector: %w", err)
		}
		service := UsecaseMessaging.NewDeadLetterUsecase(
			inspector,
			nil,
			appLogger,
			cfg.Cron.DLQInspectionBatchSize,
			0,
			"",
		)
		if err := runner.Register(scheduler.Job{
			Name:     "mq.dlq.inspect",
			Interval: time.Duration(cfg.Cron.DLQInspectionIntervalSeconds) * time.Second,
			Run:      service.InspectDeadLetters,
		}); err != nil {
			return fmt.Errorf("register dlq inspection job: %w", err)
		}
	}

	if cfg.Cron.DLQReplayEnabled {
		if cfg.Cron.DLQReplayIntervalSeconds <= 0 {
			return fmt.Errorf("cron.dlq_replay_interval_seconds must be greater than zero")
		}
		if cfg.Cron.DLQReplayBatchSize <= 0 {
			return fmt.Errorf("cron.dlq_replay_batch_size must be greater than zero")
		}
		if strings.TrimSpace(cfg.Cron.DLQReplayGroup) == "" {
			return fmt.Errorf("cron.dlq_replay_group must not be empty")
		}

		replayer, err := mq.NewDeadLetterReplayer(deadLetterOptions(cfg, cfg.Cron.DLQReplayGroup), appLogger)
		if err != nil {
			return fmt.Errorf("initialize kafka dead letter replayer: %w", err)
		}
		target, err := mq.ResolveDeadLetterReplayTarget(cfg.MQ.EventsName, cfg.Cron.DLQReplayTarget, retryLevels(cfg))
		if err != nil {
			return err
		}
		service := UsecaseMessaging.NewDeadLetterUsecase(
			nil,
			replayer,
			appLogger,
			0,
			cfg.Cron.DLQReplayBatchSize,
			target,
		)
		if err := runner.Register(scheduler.Job{
			Name:     "mq.dlq.replay",
			Interval: time.Duration(cfg.Cron.DLQReplayIntervalSeconds) * time.Second,
			Run:      service.ReplayDeadLetters,
		}); err != nil {
			return fmt.Errorf("register dlq replay job: %w", err)
		}
	}
	return nil
}

func deadLetterOptions(cfg *config.Config, groupID string) mq.DeadLetterOptions {
	return mq.DeadLetterOptions{
		KafkaOptions: mq.KafkaOptions{
			Brokers:  cfg.MQ.Kafka.Brokers,
			Topic:    mq.ResolveDeadLetterTopic(cfg.MQ.EventsName, cfg.Worker.KafkaDeadLetterTopic),
			ClientID: cfg.MQ.Kafka.ClientID,
		},
		GroupID:     strings.TrimSpace(groupID),
		MinBytes:    cfg.Worker.KafkaReadMinBytes,
		MaxBytes:    cfg.Worker.KafkaReadMaxBytes,
		MaxWait:     time.Duration(cfg.Worker.KafkaMaxWaitSeconds) * time.Second,
		PollTimeout: time.Duration(cfg.Worker.KafkaMaxWaitSeconds) * time.Second,
	}
}

func retryLevels(cfg *config.Config) []pkgkafka.RetryLevel {
	levels := make([]pkgkafka.RetryLevel, 0, len(cfg.Worker.KafkaRetryTopics))
	for _, level := range cfg.Worker.KafkaRetryTopics {
		levels = append(levels, pkgkafka.RetryLevel{Topic: level.Topic, Delay: time.Duration(level.DelaySeconds) * time.Second})
	}
	return levels
}
