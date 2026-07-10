package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domainAuth "github.com/freeDog-wy/go-backend-template/internal/domain/auth"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

func TestAuthenticateAccessToken(t *testing.T) {
	t.Parallel()

	now := time.Now()
	validSession := domainAuth.ReconstituteRefreshSession("session-1", 42, "hash", now.Add(time.Hour))

	tests := []struct {
		name         string
		tokenErr     error
		claims       *domainAuth.AccessClaims
		session      *domainAuth.RefreshSession
		sessionErr   error
		wantErr      error
		wantUserID   uint
		wantDeleteID string
	}{
		{
			name: "success when jwt and session are valid",
			claims: &domainAuth.AccessClaims{
				UserID:    42,
				SessionID: "session-1",
				Type:      "access",
			},
			session:    validSession,
			wantUserID: 42,
		},
		{
			name:     "rejects invalid jwt",
			tokenErr: errors.New("invalid jwt"),
			wantErr:  ErrInvalidAccessToken,
		},
		{
			name: "rejects missing session",
			claims: &domainAuth.AccessClaims{
				UserID:    42,
				SessionID: "session-1",
				Type:      "access",
			},
			sessionErr: shared.ErrNotFound,
			wantErr:    ErrInvalidAccessToken,
		},
		{
			name: "rejects expired session",
			claims: &domainAuth.AccessClaims{
				UserID:    42,
				SessionID: "session-1",
				Type:      "access",
			},
			session:      domainAuth.ReconstituteRefreshSession("session-1", 42, "hash", now.Add(-time.Minute)),
			wantErr:      ErrInvalidAccessToken,
			wantDeleteID: "session-1",
		},
		{
			name: "rejects mismatched session user",
			claims: &domainAuth.AccessClaims{
				UserID:    42,
				SessionID: "session-1",
				Type:      "access",
			},
			session:      domainAuth.ReconstituteRefreshSession("session-1", 99, "hash", now.Add(time.Hour)),
			wantErr:      ErrInvalidAccessToken,
			wantDeleteID: "session-1",
		},
		{
			name: "propagates session store failure",
			claims: &domainAuth.AccessClaims{
				UserID:    42,
				SessionID: "session-1",
				Type:      "access",
			},
			sessionErr: errors.New("redis unavailable"),
			wantErr:    errors.New("redis unavailable"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &stubSessionStore{
				session: tt.session,
				findErr: tt.sessionErr,
			}
			service := New(
				nil,
				nil,
				store,
				nil,
				&stubAccessTokenManager{claims: tt.claims, parseErr: tt.tokenErr},
				nil,
				nil,
				"",
				"",
				15*time.Minute,
				24*time.Hour,
			)

			identity, err := service.AuthenticateAccessToken(context.Background(), "access-token")
			if !errorIs(err, tt.wantErr) {
				t.Fatalf("AuthenticateAccessToken() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if identity == nil {
					t.Fatal("AuthenticateAccessToken() identity = nil")
				}
				if identity.UserID != tt.wantUserID {
					t.Fatalf("AuthenticateAccessToken() userID = %d, want %d", identity.UserID, tt.wantUserID)
				}
			}
			if store.deletedID != tt.wantDeleteID {
				t.Fatalf("DeleteByID() called with %q, want %q", store.deletedID, tt.wantDeleteID)
			}
		})
	}
}

type stubSessionStore struct {
	session   *domainAuth.RefreshSession
	findErr   error
	deleteErr error
	deletedID string
}

func (s *stubSessionStore) Save(context.Context, *domainAuth.RefreshSession) error {
	return nil
}

func (s *stubSessionStore) FindByID(context.Context, string) (*domainAuth.RefreshSession, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	return s.session, nil
}

func (s *stubSessionStore) FindByUserID(context.Context, uint) (*domainAuth.RefreshSession, error) {
	return nil, shared.ErrNotFound
}

func (s *stubSessionStore) DeleteByID(_ context.Context, sessionID string) error {
	s.deletedID = sessionID
	return s.deleteErr
}

func (s *stubSessionStore) DeleteByUserID(context.Context, uint) error {
	return nil
}

type stubAccessTokenManager struct {
	claims   *domainAuth.AccessClaims
	parseErr error
}

func (m *stubAccessTokenManager) IssueAccessToken(domainAuth.AccessClaims) (string, error) {
	return "", nil
}

func (m *stubAccessTokenManager) ParseAccessToken(string, time.Time) (*domainAuth.AccessClaims, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.claims, nil
}

func errorIs(err error, target error) bool {
	if target == nil {
		return err == nil
	}
	return errors.Is(err, target) || (err != nil && err.Error() == target.Error())
}
