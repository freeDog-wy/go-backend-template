package consumption

import (
	"context"
	"strings"
	"time"

	domainConsumption "github.com/freeDog-wy/go-backend-template/internal/domain/consumption"
	"github.com/freeDog-wy/go-backend-template/internal/infra/database"
	modelConsumption "github.com/freeDog-wy/go-backend-template/internal/model/consumption"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository 基于 GORM 实现消费记录表的读写。
type Repository struct {
	db *gorm.DB
}

var _ domainConsumption.Repository = (*Repository)(nil)

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Begin(ctx context.Context, command domainConsumption.BeginCommand) (domainConsumption.BeginResult, error) {
	if !command.Valid() {
		return domainConsumption.BeginResult{}, domainConsumption.ErrInvalidRecord
	}

	var result domainConsumption.BeginResult
	err := database.DB(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		model := &modelConsumption.Record{
			ConsumerGroup: command.ConsumerGroup,
			MessageKey:    command.MessageKey,
			EventName:     command.EventName,
			TraceID:       strings.TrimSpace(command.TraceID),
			Status:        string(domainConsumption.StatusProcessing),
			AttemptCount:  1,
			LockedUntil:   timeRef(command.LockedUntil),
			CreatedAt:     command.AttemptedAt,
			UpdatedAt:     command.AttemptedAt,
		}

		createResult := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "consumer_group"}, {Name: "message_key"}},
			DoNothing: true,
		}).Create(model)
		if createResult.Error != nil {
			return createResult.Error
		}
		if createResult.RowsAffected > 0 {
			result = domainConsumption.BeginResult{
				Decision:     domainConsumption.BeginDecisionAcquired,
				AttemptCount: 1,
			}
			return nil
		}

		var existing modelConsumption.Record
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("consumer_group = ? AND message_key = ?", command.ConsumerGroup, command.MessageKey).
			First(&existing).Error; err != nil {
			return err
		}

		record := existing.ToEntity()
		switch domainConsumption.Status(existing.Status) {
		case domainConsumption.StatusDone, domainConsumption.StatusDead:
			result = domainConsumption.BeginResult{
				Decision:     domainConsumption.BeginDecisionDone,
				AttemptCount: record.GetAttemptCount(),
			}
			return nil
		case domainConsumption.StatusProcessing:
			if existing.LockedUntil != nil && existing.LockedUntil.After(command.AttemptedAt) {
				result = domainConsumption.BeginResult{
					Decision:     domainConsumption.BeginDecisionLocked,
					AttemptCount: record.GetAttemptCount(),
				}
				return nil
			}
		}

		nextAttempt := existing.AttemptCount + 1
		if err := tx.Model(&modelConsumption.Record{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{
				"event_name":    command.EventName,
				"trace_id":      strings.TrimSpace(command.TraceID),
				"status":        string(domainConsumption.StatusProcessing),
				"attempt_count": nextAttempt,
				"last_error":    "",
				"locked_until":  command.LockedUntil,
				"processed_at":  nil,
				"updated_at":    command.AttemptedAt,
			}).Error; err != nil {
			return err
		}

		result = domainConsumption.BeginResult{
			Decision:     domainConsumption.BeginDecisionAcquired,
			AttemptCount: nextAttempt,
		}
		return nil
	})
	return result, err
}

func (r *Repository) MarkDone(ctx context.Context, consumerGroup, messageKey string, processedAt time.Time) error {
	if strings.TrimSpace(consumerGroup) == "" || strings.TrimSpace(messageKey) == "" || processedAt.IsZero() {
		return domainConsumption.ErrInvalidRecord
	}

	return database.DB(ctx, r.db).
		Model(&modelConsumption.Record{}).
		Where("consumer_group = ? AND message_key = ?", consumerGroup, messageKey).
		Updates(map[string]any{
			"status":       string(domainConsumption.StatusDone),
			"processed_at": processedAt,
			"locked_until": nil,
			"last_error":   "",
			"updated_at":   processedAt,
		}).Error
}

func (r *Repository) MarkFailed(ctx context.Context, consumerGroup, messageKey, lastError string, failedAt time.Time) error {
	if strings.TrimSpace(consumerGroup) == "" || strings.TrimSpace(messageKey) == "" || failedAt.IsZero() {
		return domainConsumption.ErrInvalidRecord
	}

	return database.DB(ctx, r.db).
		Model(&modelConsumption.Record{}).
		Where("consumer_group = ? AND message_key = ?", consumerGroup, messageKey).
		Updates(map[string]any{
			"status":       string(domainConsumption.StatusFailed),
			"locked_until": nil,
			"last_error":   strings.TrimSpace(lastError),
			"updated_at":   failedAt,
		}).Error
}

func (r *Repository) MarkDead(ctx context.Context, consumerGroup, messageKey, lastError string, failedAt time.Time) error {
	if strings.TrimSpace(consumerGroup) == "" || strings.TrimSpace(messageKey) == "" || failedAt.IsZero() {
		return domainConsumption.ErrInvalidRecord
	}

	return database.DB(ctx, r.db).
		Model(&modelConsumption.Record{}).
		Where("consumer_group = ? AND message_key = ?", consumerGroup, messageKey).
		Updates(map[string]any{
			"status":       string(domainConsumption.StatusDead),
			"locked_until": nil,
			"last_error":   strings.TrimSpace(lastError),
			"updated_at":   failedAt,
		}).Error
}

func timeRef(value time.Time) *time.Time {
	return &value
}
