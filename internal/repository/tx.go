package repository

import (
	"context"

	"gorm.io/gorm"
)

// TxManager 通过 context 向 Repository 传递 GORM 事务连接。
type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

type txKey struct{}

// Do 在单个 PostgreSQL 事务中执行 fn，并将事务连接写入回调 context。
func (m *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// DB 返回 context 中的事务连接；没有事务时返回默认连接。
func DB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return defaultDB
}
