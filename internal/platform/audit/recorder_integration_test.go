//go:build integration

package audit

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	baseRepository "github.com/freeDog-wy/go-backend-template/internal/repository"
	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
)

func TestRecorderIntegrationUsesCallerTransaction(t *testing.T) {
	db := testsupport.OpenPostgres(t)
	if err := db.AutoMigrate(&logModel{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	recorder := NewRecorder(New(db))
	targetID := fmt.Sprintf("audit-recorder-%d", time.Now().UnixNano())
	input := RecordInput{TargetType: "test", TargetID: targetID, Action: "record", Result: ResultSuccess}
	t.Cleanup(func() {
		_ = db.Where("target_id = ?", targetID).Delete(&logModel{}).Error
	})

	rollbackErr := errors.New("rollback")
	err := baseRepository.NewTxManager(db).Do(context.Background(), func(ctx context.Context) error {
		if err := recorder.Record(ctx, input); err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("transaction error = %v, want rollback error", err)
	}

	var count int64
	if err := db.Model(&logModel{}).Where("target_id = ?", targetID).Count(&count).Error; err != nil {
		t.Fatalf("count rolled back records: %v", err)
	}
	if count != 0 {
		t.Fatalf("rolled back records = %d, want 0", count)
	}

	if err := baseRepository.NewTxManager(db).Do(context.Background(), func(ctx context.Context) error {
		return recorder.Record(ctx, input)
	}); err != nil {
		t.Fatalf("commit record: %v", err)
	}
	if err := db.Model(&logModel{}).Where("target_id = ?", targetID).Count(&count).Error; err != nil {
		t.Fatalf("count committed records: %v", err)
	}
	if count != 1 {
		t.Fatalf("committed records = %d, want 1", count)
	}
}
