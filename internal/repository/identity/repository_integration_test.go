//go:build integration

package identity

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	modelIdentity "github.com/freeDog-wy/go-backend-template/internal/model/identity"
	"github.com/freeDog-wy/go-backend-template/internal/testkit"
)

func TestRepositoryIntegrationUserLifecycle(t *testing.T) {
	db := testkit.OpenPostgres(t)
	if err := db.AutoMigrate(&modelIdentity.User{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	repo := New(db)
	email := fmt.Sprintf("identity-it-%d@example.com", time.Now().UnixNano())
	user, _ := domainIdentity.NewUser("Alice", email)
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Unscoped().Where("id = ?", user.GetID()).Delete(&modelIdentity.User{}).Error })
	if user.GetID() == 0 {
		t.Fatal("Create() did not assign user ID")
	}
	found, err := repo.FindByEmail(context.Background(), email)
	if err != nil || found.GetID() != user.GetID() || found.GetStatus() != domainIdentity.StatusPendingVerification {
		t.Fatalf("FindByEmail() = %#v, %v", found, err)
	}
	if err := user.UpdateProfile("Alice Updated"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(context.Background(), user); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	updated, err := repo.FindByID(context.Background(), user.GetID())
	if err != nil || updated.GetName() != "Alice Updated" {
		t.Fatalf("updated user = %#v, %v", updated, err)
	}
	if err := repo.Delete(context.Background(), user.GetID()); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(context.Background(), user.GetID()); !errors.Is(err, shared.ErrNotFound) {
		t.Fatalf("FindByID() after delete error = %v", err)
	}
}
