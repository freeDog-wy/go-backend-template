package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/config"
	domainAudit "github.com/freeDog-wy/go-backend-template/internal/domain/audit"
	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"
	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
	HdlHealth "github.com/freeDog-wy/go-backend-template/internal/handler/health"
	"github.com/freeDog-wy/go-backend-template/internal/infra/database"
	"github.com/freeDog-wy/go-backend-template/internal/infra/logging"
	"github.com/freeDog-wy/go-backend-template/internal/infra/mq"
	"github.com/freeDog-wy/go-backend-template/internal/infra/tracing"
	RepoAudit "github.com/freeDog-wy/go-backend-template/internal/repository/audit"
	RepoConsumption "github.com/freeDog-wy/go-backend-template/internal/repository/consumption"
	SvcAudit "github.com/freeDog-wy/go-backend-template/internal/usecase/audit"
	SvcVerification "github.com/freeDog-wy/go-backend-template/internal/usecase/verification"
	"github.com/freeDog-wy/go-backend-template/pkg/email"
	pkgkafka "github.com/freeDog-wy/go-backend-template/pkg/kafka"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Worker struct {
	consumer    mq.Consumer
	probeServer *HdlHealth.Server
	running     atomic.Bool
	tp          *sdktrace.TracerProvider
}

func (w *Worker) Run(ctx context.Context) error {
	w.running.Store(true)
	defer w.running.Store(false)
	return w.consumer.Run(ctx)
}

func (w *Worker) ServeProbe() error {
	return w.probeServer.Serve()
}

func (w *Worker) Shutdown(ctx context.Context) error {
	err := w.probeServer.Shutdown(ctx)
	tracing.Shutdown(ctx, w.tp)
	return err
}

func initWorker(cfg *config.Config) (*Worker, error) {
	tp, err := tracing.Init(cfg.App.Mode, cfg.Tracing.Endpoint, "go-backend-template-worker")
	if err != nil {
		return nil, fmt.Errorf("initialize tracing: %w", err)
	}

	appLogger := logging.Init(cfg.App.Mode)
	db, err := database.NewPostgresDB(cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get postgres health check handle: %w", err)
	}

	if cfg.App.Mode == "production" && !strings.EqualFold(strings.TrimSpace(cfg.Email.Mode), string(email.ModeSMTP)) {
		return nil, fmt.Errorf("email.mode must be smtp in production")
	}
	emailSender, err := email.New(email.Config{
		Mode:         email.Mode(cfg.Email.Mode),
		SmtpHost:     cfg.Email.SmtpHost,
		SmtpPort:     cfg.Email.SmtpPort,
		SmtpUser:     cfg.Email.SmtpUser,
		SmtpPassword: cfg.Email.SmtpPassword,
		FromAddress:  cfg.Email.FromAddress,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize email sender: %w", err)
	}

	verificationConsumer := SvcVerification.NewConsumer(emailSender, cfg.Email.SiteBaseURL, appLogger)
	auditConsumer := SvcAudit.NewConsumer(RepoAudit.New(db), appLogger)

	retryLevels := make([]pkgkafka.RetryLevel, 0, len(cfg.Worker.KafkaRetryTopics))
	for _, level := range cfg.Worker.KafkaRetryTopics {
		retryLevels = append(retryLevels, pkgkafka.RetryLevel{Topic: level.Topic, Delay: time.Duration(level.DelaySeconds) * time.Second})
	}
	consumer, err := mq.NewConsumer(mq.ConsumerOptions{
		KafkaOptions:      mq.KafkaOptions{Brokers: cfg.MQ.Kafka.Brokers, Topic: cfg.MQ.EventsName, ClientID: cfg.MQ.Kafka.ClientID},
		GroupID:           cfg.Worker.ConsumerGroup,
		MaxRetries:        cfg.Worker.ConsumerMaxRetries,
		ProcessingLockTTL: time.Duration(cfg.Worker.ConsumerProcessingLockSeconds) * time.Second,
		MinBytes:          cfg.Worker.KafkaReadMinBytes,
		MaxBytes:          cfg.Worker.KafkaReadMaxBytes,
		MaxWait:           time.Duration(cfg.Worker.KafkaMaxWaitSeconds) * time.Second,
		RetryLevels:       retryLevels,
		DeadLetterTopic:   mq.ResolveDeadLetterTopic(cfg.MQ.EventsName, cfg.Worker.KafkaDeadLetterTopic),
	}, RepoConsumption.New(db), appLogger)
	if err != nil {
		return nil, fmt.Errorf("initialize kafka consumer: %w", err)
	}
	worker := &Worker{
		consumer: consumer,
		tp:       tp,
	}
	worker.probeServer = HdlHealth.NewServer(cfg.Worker.Probe.Address(), map[string]HdlHealth.Checker{
		"consumer": HdlHealth.CheckFunc(func(context.Context) error {
			if !worker.running.Load() {
				return errors.New("consumer loop is not running")
			}
			return nil
		}),
		"database": HdlHealth.CheckFunc(sqlDB.PingContext),
		"kafka": HdlHealth.CheckFunc(func(ctx context.Context) error {
			return mq.PingKafka(ctx, cfg.MQ.Kafka.Brokers)
		}),
	}, 2*time.Second)

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

	return worker, nil
}
