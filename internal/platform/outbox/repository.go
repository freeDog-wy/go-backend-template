package outbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	repositorytx "github.com/freeDog-wy/go-backend-template/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, events ...*Event) error {
	if len(events) == 0 {
		return nil
	}

	models := make([]*eventModel, 0, len(events))
	for _, event := range events {
		models = append(models, eventModelFromEvent(event))
	}

	return repositorytx.DB(ctx, r.db).Create(&models).Error
}

func (r *Repository) ListUnpublished(ctx context.Context, limit int) ([]*Event, error) {
	if limit <= 0 {
		limit = 100
	}

	var models []*eventModel
	if err := repositorytx.DB(ctx, r.db).
		Where("published_at IS NULL").
		Order("id ASC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, err
	}

	events := make([]*Event, 0, len(models))
	for _, model := range models {
		events = append(events, model.toEvent())
	}
	return events, nil
}

// ClaimUnpublished assigns a short-lived lease to a batch of unpublished events.
// The transaction only covers claiming rows; callers must publish outside it.
func (r *Repository) ClaimUnpublished(ctx context.Context, claimant string, now time.Time, leaseTTL time.Duration, limit int) ([]*Event, error) {
	claimant = strings.TrimSpace(claimant)
	if claimant == "" {
		return nil, fmt.Errorf("outbox claimant must not be empty")
	}
	if now.IsZero() {
		return nil, fmt.Errorf("outbox claim time must not be zero")
	}
	if leaseTTL <= 0 {
		return nil, fmt.Errorf("outbox claim lease must be greater than zero")
	}
	if limit <= 0 {
		limit = 100
	}

	models := make([]*eventModel, 0, limit)
	claimUntil := now.Add(leaseTTL)
	err := repositorytx.DB(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("published_at IS NULL AND (claim_until IS NULL OR claim_until <= ?)", now).
			Order("id ASC").
			Limit(limit).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Find(&models).Error; err != nil {
			return err
		}
		if len(models) == 0 {
			return nil
		}

		ids := make([]uint, 0, len(models))
		for _, model := range models {
			ids = append(ids, model.ID)
		}
		return tx.Model(&eventModel{}).
			Where("id IN ? AND published_at IS NULL", ids).
			Updates(map[string]any{"claimed_by": claimant, "claim_until": claimUntil}).Error
	})
	if err != nil {
		return nil, err
	}

	events := make([]*Event, 0, len(models))
	for _, model := range models {
		events = append(events, model.toEvent())
	}
	return events, nil
}

// MarkPublished marks an event only when it remains owned by claimant.
func (r *Repository) MarkPublished(ctx context.Context, id uint, claimant string, publishedAt time.Time) (bool, error) {
	if id == 0 {
		return false, fmt.Errorf("outbox event ID must not be zero")
	}
	claimant = strings.TrimSpace(claimant)
	if claimant == "" {
		return false, fmt.Errorf("outbox claimant must not be empty")
	}
	if publishedAt.IsZero() {
		return false, fmt.Errorf("outbox published time must not be zero")
	}

	result := repositorytx.DB(ctx, r.db).
		Model(&eventModel{}).
		Where("id = ? AND published_at IS NULL AND claimed_by = ? AND claim_until > ?", id, claimant, publishedAt).
		Updates(map[string]any{"published_at": publishedAt, "claimed_by": "", "claim_until": nil})
	return result.RowsAffected == 1, result.Error
}

// ReleaseClaims makes unprocessed events available for another publisher.
func (r *Repository) ReleaseClaims(ctx context.Context, ids []uint, claimant string) error {
	if len(ids) == 0 {
		return nil
	}
	claimant = strings.TrimSpace(claimant)
	if claimant == "" {
		return fmt.Errorf("outbox claimant must not be empty")
	}

	return repositorytx.DB(ctx, r.db).
		Model(&eventModel{}).
		Where("id IN ? AND published_at IS NULL AND claimed_by = ?", ids, claimant).
		Updates(map[string]any{"claimed_by": "", "claim_until": nil}).Error
}
