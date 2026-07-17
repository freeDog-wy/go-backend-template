package audit

import (
	"context"

	repositorytx "github.com/freeDog-wy/go-backend-template/internal/repository"

	"gorm.io/gorm"
)

// Writer 持久化审计记录。
type Writer interface {
	Create(ctx context.Context, log *AuditLog) error
}

// Store 基于 GORM 持久化审计记录。
type Store struct {
	db *gorm.DB
}

var _ Writer = (*Store)(nil)

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) g(ctx context.Context) gorm.Interface[logModel] {
	return gorm.G[logModel](repositorytx.DB(ctx, s.db))
}

func (s *Store) Create(ctx context.Context, log *AuditLog) error {
	return s.g(ctx).Create(ctx, logModelFromLog(log))
}
