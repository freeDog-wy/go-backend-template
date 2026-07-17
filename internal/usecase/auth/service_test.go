package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domainAuth "github.com/freeDog-wy/go-backend-template/internal/domain/auth"
	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	platformAudit "github.com/freeDog-wy/go-backend-template/internal/platform/audit"
)

func TestParseAccessToken(t *testing.T) {
	t.Parallel()

	t.Run("returns identity when jwt is valid", func(t *testing.T) {
		t.Parallel()

		service := New(
			nil,
			nil,
			nil,
			nil,
			&stubAccessTokenManager{
				claims: &domainAuth.AccessClaims{
					UserID:    42,
					SessionID: "session-1",
					Type:      "access",
				},
			},
			nil,
			nil,
			"",
			"",
			15*time.Minute,
			24*time.Hour,
		)

		identity, err := service.ParseAccessToken("access-token")
		if err != nil {
			t.Fatalf("ParseAccessToken() error = %v", err)
		}
		if identity == nil {
			t.Fatal("ParseAccessToken() identity = nil")
		}
		if identity.UserID != 42 || identity.SessionID != "session-1" {
			t.Fatalf("ParseAccessToken() identity = %+v", identity)
		}
	})

	t.Run("returns invalid access token when parser fails", func(t *testing.T) {
		t.Parallel()

		service := New(
			nil,
			nil,
			nil,
			nil,
			&stubAccessTokenManager{parseErr: errors.New("bad jwt")},
			nil,
			nil,
			"",
			"",
			15*time.Minute,
			24*time.Hour,
		)

		identity, err := service.ParseAccessToken("access-token")
		if !errors.Is(err, ErrInvalidAccessToken) {
			t.Fatalf("ParseAccessToken() error = %v, want %v", err, ErrInvalidAccessToken)
		}
		if identity != nil {
			t.Fatalf("ParseAccessToken() identity = %+v, want nil", identity)
		}
	})
}

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

func TestRefresh(t *testing.T) {
	t.Parallel()

	now := time.Now()
	validSecret := "refresh-secret"
	validHash := hashToken(validSecret)
	validSession := domainAuth.ReconstituteRefreshSession("session-1", 42, validHash, now.Add(time.Hour))

	t.Run("returns invalid refresh token for malformed token", func(t *testing.T) {
		t.Parallel()

		service := New(nil, nil, &stubSessionStore{}, nil, &stubAccessTokenManager{}, nil, nil, "", "", 15*time.Minute, 24*time.Hour)

		result, err := service.Refresh(context.Background(), RefreshCmd{RefreshToken: "bad-token"})
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("Refresh() error = %v, want %v", err, ErrInvalidRefreshToken)
		}
		if result != nil {
			t.Fatalf("Refresh() result = %+v, want nil", result)
		}
	})

	t.Run("returns invalid refresh token when session is missing", func(t *testing.T) {
		t.Parallel()

		store := &stubSessionStore{findErr: shared.ErrNotFound}
		service := New(nil, nil, store, nil, &stubAccessTokenManager{}, nil, nil, "", "", 15*time.Minute, 24*time.Hour)

		result, err := service.Refresh(context.Background(), RefreshCmd{RefreshToken: "session-1." + validSecret})
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("Refresh() error = %v, want %v", err, ErrInvalidRefreshToken)
		}
		if result != nil {
			t.Fatalf("Refresh() result = %+v, want nil", result)
		}
	})

	t.Run("deletes expired or mismatched session and rejects token", func(t *testing.T) {
		t.Parallel()

		store := &stubSessionStore{
			session: domainAuth.ReconstituteRefreshSession("session-1", 42, "other-hash", now.Add(time.Hour)),
		}
		service := New(nil, nil, store, nil, &stubAccessTokenManager{}, nil, nil, "", "", 15*time.Minute, 24*time.Hour)

		result, err := service.Refresh(context.Background(), RefreshCmd{RefreshToken: "session-1." + validSecret})
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("Refresh() error = %v, want %v", err, ErrInvalidRefreshToken)
		}
		if result != nil {
			t.Fatalf("Refresh() result = %+v, want nil", result)
		}
		if store.deletedID != "session-1" {
			t.Fatalf("DeleteByID() called with %q, want %q", store.deletedID, "session-1")
		}
	})

	t.Run("returns invalid refresh token when user is missing", func(t *testing.T) {
		t.Parallel()

		store := &stubSessionStore{session: validSession}
		userRepo := &stubIdentityRepo{findByIDErr: shared.ErrNotFound}
		service := New(userRepo, nil, store, nil, &stubAccessTokenManager{}, nil, nil, "", "", 15*time.Minute, 24*time.Hour)

		result, err := service.Refresh(context.Background(), RefreshCmd{RefreshToken: "session-1." + validSecret})
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("Refresh() error = %v, want %v", err, ErrInvalidRefreshToken)
		}
		if result != nil {
			t.Fatalf("Refresh() result = %+v, want nil", result)
		}
	})

	t.Run("returns user status error when user is not allowed to login", func(t *testing.T) {
		t.Parallel()

		store := &stubSessionStore{session: validSession}
		userRepo := &stubIdentityRepo{userByID: newTestUser(42, domainIdentity.StatusLocked, true)}
		service := New(userRepo, nil, store, nil, &stubAccessTokenManager{}, nil, nil, "", "", 15*time.Minute, 24*time.Hour)

		result, err := service.Refresh(context.Background(), RefreshCmd{RefreshToken: "session-1." + validSecret})
		if !errors.Is(err, ErrUserLocked) {
			t.Fatalf("Refresh() error = %v, want %v", err, ErrUserLocked)
		}
		if result != nil {
			t.Fatalf("Refresh() result = %+v, want nil", result)
		}
	})

	t.Run("issues new tokens and invalidates old session on success", func(t *testing.T) {
		t.Parallel()

		store := &stubSessionStore{session: validSession}
		userRepo := &stubIdentityRepo{userByID: newTestUser(42, domainIdentity.StatusActive, true)}
		tokenManager := &stubAccessTokenManager{issueToken: "new-access-token"}
		service := New(userRepo, nil, store, nil, tokenManager, nil, nil, "issuer", "audience", 15*time.Minute, 24*time.Hour)

		result, err := service.Refresh(context.Background(), RefreshCmd{RefreshToken: "session-1." + validSecret})
		if err != nil {
			t.Fatalf("Refresh() error = %v", err)
		}
		if result == nil {
			t.Fatal("Refresh() result = nil")
		}
		if result.AccessToken != "new-access-token" {
			t.Fatalf("Refresh() access token = %q, want %q", result.AccessToken, "new-access-token")
		}
		if result.User == nil || result.User.ID != 42 {
			t.Fatalf("Refresh() user = %+v", result.User)
		}
		if store.deletedID != "session-1" {
			t.Fatalf("DeleteByID() called with %q, want %q", store.deletedID, "session-1")
		}
		if store.deletedUserID != 42 {
			t.Fatalf("DeleteByUserID() called with %d, want %d", store.deletedUserID, 42)
		}
		if store.savedSession == nil {
			t.Fatal("Save() was not called")
		}
		refreshSessionID, refreshSecret, parseErr := parseRefreshToken(result.RefreshToken)
		if parseErr != nil {
			t.Fatalf("parseRefreshToken() error = %v", parseErr)
		}
		if refreshSessionID != store.savedSession.GetID() {
			t.Fatalf("refresh session id = %q, want %q", refreshSessionID, store.savedSession.GetID())
		}
		if hashToken(refreshSecret) != store.savedSession.GetTokenHash() {
			t.Fatal("saved refresh session token hash does not match returned refresh token")
		}
		if tokenManager.issuedClaims == nil {
			t.Fatal("IssueAccessToken() was not called")
		}
		if tokenManager.issuedClaims.UserID != 42 {
			t.Fatalf("issued claims user id = %d, want %d", tokenManager.issuedClaims.UserID, 42)
		}
		if tokenManager.issuedClaims.SessionID != store.savedSession.GetID() {
			t.Fatalf("issued claims session id = %q, want %q", tokenManager.issuedClaims.SessionID, store.savedSession.GetID())
		}
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()

	t.Run("returns invalid refresh token for malformed token", func(t *testing.T) {
		t.Parallel()

		service := New(nil, nil, &stubSessionStore{}, nil, nil, nil, nil, "", "", 15*time.Minute, 24*time.Hour)
		err := service.Logout(context.Background(), LogoutCmd{RefreshToken: "bad-token"})
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("Logout() error = %v, want %v", err, ErrInvalidRefreshToken)
		}
	})

	t.Run("returns invalid refresh token when session is missing", func(t *testing.T) {
		t.Parallel()

		store := &stubSessionStore{findErr: shared.ErrNotFound}
		service := New(nil, nil, store, nil, nil, nil, nil, "", "", 15*time.Minute, 24*time.Hour)
		err := service.Logout(context.Background(), LogoutCmd{RefreshToken: "session-1.secret"})
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("Logout() error = %v, want %v", err, ErrInvalidRefreshToken)
		}
	})

	t.Run("returns delete error when session removal fails", func(t *testing.T) {
		t.Parallel()

		store := &stubSessionStore{
			session:   domainAuth.ReconstituteRefreshSession("session-1", 42, "hash", time.Now().Add(time.Hour)),
			deleteErr: errors.New("redis unavailable"),
		}
		service := New(nil, nil, store, nil, nil, nil, nil, "", "", 15*time.Minute, 24*time.Hour)
		err := service.Logout(context.Background(), LogoutCmd{RefreshToken: "session-1.secret"})
		if err == nil || err.Error() != "redis unavailable" {
			t.Fatalf("Logout() error = %v, want redis unavailable", err)
		}
	})

	t.Run("deletes session and publishes audit log on success", func(t *testing.T) {
		t.Parallel()

		store := &stubSessionStore{
			session: domainAuth.ReconstituteRefreshSession("session-1", 42, "hash", time.Now().Add(time.Hour)),
		}
		eventBus := &stubEventBus{}
		service := New(nil, nil, store, nil, nil, eventBus, nil, "", "", 15*time.Minute, 24*time.Hour)

		err := service.Logout(context.Background(), LogoutCmd{
			RefreshToken: "session-1.secret",
			IP:           "127.0.0.1",
			UserAgent:    "unit-test",
		})
		if err != nil {
			t.Fatalf("Logout() error = %v", err)
		}
		if store.deletedID != "session-1" {
			t.Fatalf("DeleteByID() called with %q, want %q", store.deletedID, "session-1")
		}
		assertSingleAuditEvent(t, eventBus, func(event platformAudit.LogRequested) {
			if event.Action != auditActionLogout {
				t.Fatalf("audit action = %q, want %q", event.Action, auditActionLogout)
			}
			if event.Result != platformAudit.ResultSuccess {
				t.Fatalf("audit result = %q, want %q", event.Result, platformAudit.ResultSuccess)
			}
			if event.ActorUserID == nil || *event.ActorUserID != 42 {
				t.Fatalf("audit actor = %v, want 42", event.ActorUserID)
			}
		})
	})
}

func TestChangePassword(t *testing.T) {
	t.Parallel()

	now := time.Now()
	credential := domainAuth.ReconstituteUserCredential(42, "old-hash", now.Add(-time.Hour), now.Add(-2*time.Hour), now.Add(-time.Hour))

	t.Run("returns invalid current password when credential is missing", func(t *testing.T) {
		t.Parallel()

		credentialRepo := &stubCredentialRepo{findErr: shared.ErrNotFound}
		service := New(nil, credentialRepo, &stubSessionStore{}, &stubPasswordHasher{}, nil, nil, nil, "", "", 15*time.Minute, 24*time.Hour)

		err := service.ChangePassword(context.Background(), ChangePasswordCmd{
			UserID:          42,
			CurrentPassword: "old",
			NewPassword:     "new",
		})
		if !errors.Is(err, ErrInvalidCurrentPassword) {
			t.Fatalf("ChangePassword() error = %v, want %v", err, ErrInvalidCurrentPassword)
		}
	})

	t.Run("publishes failure audit when current password is invalid", func(t *testing.T) {
		t.Parallel()

		credentialRepo := &stubCredentialRepo{credential: credential}
		passwordHasher := &stubPasswordHasher{verifyResult: false}
		eventBus := &stubEventBus{}
		service := New(nil, credentialRepo, &stubSessionStore{}, passwordHasher, nil, eventBus, nil, "", "", 15*time.Minute, 24*time.Hour)

		err := service.ChangePassword(context.Background(), ChangePasswordCmd{
			UserID:          42,
			CurrentPassword: "wrong-password",
			NewPassword:     "new-password",
		})
		if !errors.Is(err, ErrInvalidCurrentPassword) {
			t.Fatalf("ChangePassword() error = %v, want %v", err, ErrInvalidCurrentPassword)
		}
		assertSingleAuditEvent(t, eventBus, func(event platformAudit.LogRequested) {
			if event.Action != auditActionChangePassword {
				t.Fatalf("audit action = %q, want %q", event.Action, auditActionChangePassword)
			}
			if event.Result != platformAudit.ResultFailure {
				t.Fatalf("audit result = %q, want %q", event.Result, platformAudit.ResultFailure)
			}
			if event.Metadata["reason"] != "invalid_current_password" {
				t.Fatalf("audit metadata reason = %v", event.Metadata["reason"])
			}
		})
	})

	t.Run("returns hash error", func(t *testing.T) {
		t.Parallel()

		credentialRepo := &stubCredentialRepo{credential: credential}
		passwordHasher := &stubPasswordHasher{
			verifyResult: true,
			hashErr:      errors.New("bcrypt unavailable"),
		}
		service := New(nil, credentialRepo, &stubSessionStore{}, passwordHasher, nil, nil, nil, "", "", 15*time.Minute, 24*time.Hour)

		err := service.ChangePassword(context.Background(), ChangePasswordCmd{
			UserID:          42,
			CurrentPassword: "old-password",
			NewPassword:     "new-password",
		})
		if err == nil || err.Error() != "bcrypt unavailable" {
			t.Fatalf("ChangePassword() error = %v, want bcrypt unavailable", err)
		}
	})

	t.Run("returns repository update error", func(t *testing.T) {
		t.Parallel()

		credentialRepo := &stubCredentialRepo{
			credential: credential,
			updateErr:  errors.New("db update failed"),
		}
		passwordHasher := &stubPasswordHasher{
			verifyResult: true,
			hashValue:    "new-hash",
		}
		service := New(nil, credentialRepo, &stubSessionStore{}, passwordHasher, nil, nil, nil, "", "", 15*time.Minute, 24*time.Hour)

		err := service.ChangePassword(context.Background(), ChangePasswordCmd{
			UserID:          42,
			CurrentPassword: "old-password",
			NewPassword:     "new-password",
		})
		if err == nil || err.Error() != "db update failed" {
			t.Fatalf("ChangePassword() error = %v, want db update failed", err)
		}
	})

	t.Run("updates credential, invalidates sessions, and publishes success audit", func(t *testing.T) {
		t.Parallel()

		credentialRepo := &stubCredentialRepo{credential: credential}
		store := &stubSessionStore{deleteByUserErr: errors.New("redis unavailable")}
		passwordHasher := &stubPasswordHasher{
			verifyResult: true,
			hashValue:    "new-hash",
		}
		eventBus := &stubEventBus{}
		service := New(nil, credentialRepo, store, passwordHasher, nil, eventBus, nil, "", "", 15*time.Minute, 24*time.Hour)

		err := service.ChangePassword(context.Background(), ChangePasswordCmd{
			UserID:          42,
			CurrentPassword: "old-password",
			NewPassword:     "new-password",
			IP:              "127.0.0.1",
			UserAgent:       "unit-test",
		})
		if err != nil {
			t.Fatalf("ChangePassword() error = %v", err)
		}
		if credentialRepo.updatedCredential == nil {
			t.Fatal("Update() was not called")
		}
		if credentialRepo.updatedCredential.GetPasswordHash() != "new-hash" {
			t.Fatalf("updated password hash = %q, want %q", credentialRepo.updatedCredential.GetPasswordHash(), "new-hash")
		}
		if store.deletedUserID != 42 {
			t.Fatalf("DeleteByUserID() called with %d, want %d", store.deletedUserID, 42)
		}
		assertSingleAuditEvent(t, eventBus, func(event platformAudit.LogRequested) {
			if event.Action != auditActionChangePassword {
				t.Fatalf("audit action = %q, want %q", event.Action, auditActionChangePassword)
			}
			if event.Result != platformAudit.ResultSuccess {
				t.Fatalf("audit result = %q, want %q", event.Result, platformAudit.ResultSuccess)
			}
		})
	})
}

type stubSessionStore struct {
	session         *domainAuth.RefreshSession
	findErr         error
	saveErr         error
	deleteErr       error
	deleteByUserErr error
	deletedID       string
	deletedUserID   uint
	savedSession    *domainAuth.RefreshSession
}

func (s *stubSessionStore) Save(_ context.Context, session *domainAuth.RefreshSession) error {
	s.savedSession = session
	return s.saveErr
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

func (s *stubSessionStore) DeleteByUserID(_ context.Context, userID uint) error {
	s.deletedUserID = userID
	return s.deleteByUserErr
}

type stubAccessTokenManager struct {
	claims       *domainAuth.AccessClaims
	parseErr     error
	issueToken   string
	issueErr     error
	issuedClaims *domainAuth.AccessClaims
}

func (m *stubAccessTokenManager) IssueAccessToken(claims domainAuth.AccessClaims) (string, error) {
	m.issuedClaims = &claims
	if m.issueErr != nil {
		return "", m.issueErr
	}
	if m.issueToken != "" {
		return m.issueToken, nil
	}
	return "issued-access-token", nil
}

func (m *stubAccessTokenManager) ParseAccessToken(string, time.Time) (*domainAuth.AccessClaims, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.claims, nil
}

type stubIdentityRepo struct {
	userByID       *domainIdentity.User
	userByEmail    *domainIdentity.User
	findByIDErr    error
	findByEmailErr error
	updateErr      error
	updatedUser    *domainIdentity.User
}

func (r *stubIdentityRepo) FindByID(context.Context, uint) (*domainIdentity.User, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	return r.userByID, nil
}

func (r *stubIdentityRepo) FindByEmail(context.Context, string) (*domainIdentity.User, error) {
	if r.findByEmailErr != nil {
		return nil, r.findByEmailErr
	}
	return r.userByEmail, nil
}

func (r *stubIdentityRepo) List(context.Context, shared.PageQuery) ([]*domainIdentity.User, int64, error) {
	return nil, 0, nil
}

func (r *stubIdentityRepo) Create(context.Context, *domainIdentity.User) error {
	return nil
}

func (r *stubIdentityRepo) Update(_ context.Context, user *domainIdentity.User) error {
	r.updatedUser = user
	return r.updateErr
}

func (r *stubIdentityRepo) Delete(context.Context, uint) error {
	return nil
}

type stubCredentialRepo struct {
	credential        *domainAuth.UserCredential
	findErr           error
	updateErr         error
	updatedCredential *domainAuth.UserCredential
}

func (r *stubCredentialRepo) Create(context.Context, *domainAuth.UserCredential) error {
	return nil
}

func (r *stubCredentialRepo) FindByUserID(context.Context, uint) (*domainAuth.UserCredential, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.credential, nil
}

func (r *stubCredentialRepo) Update(_ context.Context, credential *domainAuth.UserCredential) error {
	r.updatedCredential = credential
	return r.updateErr
}

type stubPasswordHasher struct {
	verifyResult bool
	hashValue    string
	hashErr      error
}

func (h *stubPasswordHasher) Hash(string) (string, error) {
	if h.hashErr != nil {
		return "", h.hashErr
	}
	if h.hashValue != "" {
		return h.hashValue, nil
	}
	return "hashed-password", nil
}

func (h *stubPasswordHasher) Verify(string, string) bool {
	return h.verifyResult
}

type stubEventBus struct {
	published  []shared.Event
	publishErr error
}

func (b *stubEventBus) Publish(_ context.Context, events ...shared.Event) error {
	b.published = append(b.published, events...)
	return b.publishErr
}

func assertSingleAuditEvent(t *testing.T, bus *stubEventBus, assertFn func(platformAudit.LogRequested)) {
	t.Helper()

	if len(bus.published) != 1 {
		t.Fatalf("published events = %d, want 1", len(bus.published))
	}

	event, ok := bus.published[0].(platformAudit.LogRequested)
	if !ok {
		t.Fatalf("published event type = %T, want platformAudit.LogRequested", bus.published[0])
	}
	assertFn(event)
}

func newTestUser(id uint, status domainIdentity.Status, emailVerified bool) *domainIdentity.User {
	now := time.Now()
	return domainIdentity.ReconstituteUser(
		id,
		"Test User",
		"user@example.com",
		status,
		emailVerified,
		time.Time{},
		now.Add(-2*time.Hour),
		now.Add(-time.Hour),
		nil,
	)
}

func errorIs(err error, target error) bool {
	if target == nil {
		return err == nil
	}
	return errors.Is(err, target) || (err != nil && err.Error() == target.Error())
}
