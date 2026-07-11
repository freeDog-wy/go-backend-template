//go:build integration

package verification

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
	modelVerification "github.com/freeDog-wy/go-backend-template/internal/model/verification"
	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
)

func TestRepositoryIntegrationEmailTokenLifecycle(t *testing.T) {
	db := testsupport.OpenPostgres(t)
	if err := db.AutoMigrate(&modelVerification.EmailVerificationToken{}, &modelVerification.PasswordResetToken{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	repo := New(db)
	now := time.Now().UTC()
	hash := fmt.Sprintf("email-token-%d", now.UnixNano())
	token, _ := domainVerification.NewEmailVerificationToken(42, hash, now.Add(time.Hour))
	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Where("token_hash = ?", hash).Delete(&modelVerification.EmailVerificationToken{}).Error })
	active, err := repo.FindActiveByTokenHash(context.Background(), hash, now)
	if err != nil || active.GetUserID() != 42 {
		t.Fatalf("FindActiveByTokenHash() = %#v, %v", active, err)
	}
	if err := repo.InvalidateByUserID(context.Background(), 42, now); err != nil {
		t.Fatalf("InvalidateByUserID() error = %v", err)
	}
	if _, err := repo.FindActiveByTokenHash(context.Background(), hash, now); !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("invalidated token error = %v", err)
	}
	expiredHash := fmt.Sprintf("expired-token-%d", now.UnixNano())
	expired, _ := domainVerification.NewEmailVerificationToken(43, expiredHash, now.Add(-time.Hour))
	if err := repo.Create(context.Background(), expired); err != nil {
		t.Fatalf("create expired token: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Where("token_hash = ?", expiredHash).Delete(&modelVerification.EmailVerificationToken{}).Error
	})
	if _, err := repo.FindActiveByTokenHash(context.Background(), expiredHash, now); !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("expired token error = %v", err)
	}
}
