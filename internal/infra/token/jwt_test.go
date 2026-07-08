package token

import (
	"testing"
	"time"

	domainAuth "github.com/freeDog-wy/go-backend-template/internal/domain/auth"
)

func TestJWTManagerIssueAndParse(t *testing.T) {
	t.Parallel()

	manager := NewJWTManager("issuer", "audience", "secret")
	now := time.Unix(1_700_000_000, 0)

	token, err := manager.IssueAccessToken(domainAuth.AccessClaims{
		UserID:    42,
		SessionID: "session-1",
		Type:      "access",
		Issuer:    "issuer",
		Audience:  "audience",
		IssuedAt:  now,
		ExpiresAt: now.Add(15 * time.Minute),
	})
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	claims, err := manager.ParseAccessToken(token, now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("claims.UserID = %d, want 42", claims.UserID)
	}
	if claims.SessionID != "session-1" {
		t.Fatalf("claims.SessionID = %q, want session-1", claims.SessionID)
	}
}

func TestJWTManagerRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	manager := NewJWTManager("issuer", "audience", "secret")
	now := time.Unix(1_700_000_000, 0)

	token, err := manager.IssueAccessToken(domainAuth.AccessClaims{
		UserID:    7,
		SessionID: "session-2",
		Type:      "access",
		Issuer:    "issuer",
		Audience:  "audience",
		IssuedAt:  now,
		ExpiresAt: now.Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("IssueAccessToken() error = %v", err)
	}

	if _, err := manager.ParseAccessToken(token, now.Add(2*time.Minute)); err == nil {
		t.Fatal("ParseAccessToken() expected error for expired token")
	}
}
