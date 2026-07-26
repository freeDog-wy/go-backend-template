package identity

import (
	"testing"
	"time"
)

func TestRecordFailedLoginLocksAtThreshold(t *testing.T) {
	t.Parallel()

	user := ReconstituteUserWithLoginFailures(
		42,
		"Test User",
		"user@example.com",
		StatusActive,
		true,
		1,
		time.Time{},
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-time.Hour),
		nil,
	)

	locked, err := user.RecordFailedLogin(2)
	if err != nil {
		t.Fatalf("RecordFailedLogin() error = %v", err)
	}
	if !locked {
		t.Fatal("RecordFailedLogin() locked = false, want true")
	}
	if !user.IsLocked() {
		t.Fatal("user should be locked after reaching the threshold")
	}
	if user.GetFailedLoginAttempts() != 2 {
		t.Fatalf("failed login attempts = %d, want 2", user.GetFailedLoginAttempts())
	}
}

func TestActivateUnlocksUserAndResetsFailedAttempts(t *testing.T) {
	t.Parallel()

	user := ReconstituteUserWithLoginFailures(
		42,
		"Test User",
		"user@example.com",
		StatusLocked,
		true,
		5,
		time.Time{},
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-time.Hour),
		nil,
	)

	if err := user.Activate(); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if !user.IsActive() {
		t.Fatal("user should be active after unlock")
	}
	if user.GetFailedLoginAttempts() != 0 {
		t.Fatalf("failed login attempts = %d, want 0", user.GetFailedLoginAttempts())
	}
}
