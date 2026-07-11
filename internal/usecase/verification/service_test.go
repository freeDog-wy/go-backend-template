package verification

import (
	"context"
	"errors"
	"testing"
	"time"

	domainAuth "github.com/freeDog-wy/go-backend-template/internal/domain/auth"
	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
)

func TestVerifyEmail(t *testing.T) {
	t.Parallel()
	now := time.Now()

	t.Run("activates the pending user consumes token and publishes audit event", func(t *testing.T) {
		t.Parallel()
		user, _ := domainIdentity.NewUser("Alice", "alice@example.com")
		token := domainVerification.ReconstituteEmailVerificationToken(1, 7, hashToken("token"), now.Add(time.Hour), nil, now)
		users := &verificationUserRepo{user: user}
		tokens := &verificationRepo{emailToken: token}
		bus := &verificationBus{}
		service := New(&verificationTx{}, users, tokens, nil, nil, nil, bus, nil)

		err := service.VerifyEmail(context.Background(), VerifyEmailCmd{Token: "token", IP: "127.0.0.1"})
		if err != nil {
			t.Fatalf("VerifyEmail() error = %v", err)
		}
		if !user.IsActive() || !user.IsEmailVerified() || tokens.updatedEmail != token || token.GetConsumedAt() == nil {
			t.Fatal("email verification state was not persisted")
		}
		if len(bus.events) != 1 || bus.events[0].EventName() != "audit.log.requested" {
			t.Fatalf("audit events = %#v", bus.events)
		}
	})

	t.Run("maps missing token to invalid token", func(t *testing.T) {
		t.Parallel()
		service := New(&verificationTx{}, &verificationUserRepo{}, &verificationRepo{emailErr: shared.ErrNotFound}, nil, nil, nil, nil, nil)
		if err := service.VerifyEmail(context.Background(), VerifyEmailCmd{Token: "missing"}); !errors.Is(err, ErrInvalidVerificationToken) {
			t.Fatalf("VerifyEmail() error = %v", err)
		}
	})
}

func TestResetPassword(t *testing.T) {
	t.Parallel()
	now := time.Now()
	credential := domainAuth.ReconstituteUserCredential(7, "old", now, now, now)
	token := domainVerification.ReconstitutePasswordResetToken(1, 7, hashToken("reset"), now.Add(time.Hour), nil, now)
	creds := &verificationCredentialRepo{credential: credential}
	tokens := &verificationRepo{resetToken: token}
	sessions := &verificationSessionStore{}
	bus := &verificationBus{}
	service := New(&verificationTx{}, &verificationUserRepo{}, tokens, creds, &verificationHasher{}, sessions, bus, nil)

	err := service.ResetPassword(context.Background(), ResetPasswordCmd{Token: "reset", Password: "new-password"})
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if credential.GetPasswordHash() != "hashed:new-password" || creds.updated != credential || tokens.updatedReset != token || token.GetConsumedAt() == nil {
		t.Fatal("password reset changes were not persisted")
	}
	if sessions.deletedUserID != 7 || len(bus.events) != 1 || bus.events[0].EventName() != "audit.log.requested" {
		t.Fatalf("session/audit state: deleted=%d events=%d", sessions.deletedUserID, len(bus.events))
	}
}

type verificationTx struct{}

func (*verificationTx) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type verificationUserRepo struct{ user *domainIdentity.User }

func (r *verificationUserRepo) FindByID(context.Context, uint) (*domainIdentity.User, error) {
	if r.user == nil {
		return nil, shared.ErrNotFound
	}
	return r.user, nil
}
func (*verificationUserRepo) FindByEmail(context.Context, string) (*domainIdentity.User, error) {
	return nil, shared.ErrNotFound
}
func (*verificationUserRepo) List(context.Context, shared.PageQuery) ([]*domainIdentity.User, int64, error) {
	return nil, 0, nil
}
func (*verificationUserRepo) Create(context.Context, *domainIdentity.User) error { return nil }
func (*verificationUserRepo) Update(context.Context, *domainIdentity.User) error { return nil }
func (*verificationUserRepo) Delete(context.Context, uint) error                 { return nil }

type verificationRepo struct {
	emailToken   *domainVerification.EmailVerificationToken
	emailErr     error
	resetToken   *domainVerification.PasswordResetToken
	updatedEmail *domainVerification.EmailVerificationToken
	updatedReset *domainVerification.PasswordResetToken
}

func (*verificationRepo) Create(context.Context, *domainVerification.EmailVerificationToken) error {
	return nil
}
func (r *verificationRepo) FindActiveByTokenHash(context.Context, string, time.Time) (*domainVerification.EmailVerificationToken, error) {
	if r.emailErr != nil {
		return nil, r.emailErr
	}
	return r.emailToken, nil
}
func (*verificationRepo) InvalidateByUserID(context.Context, uint, time.Time) error { return nil }
func (r *verificationRepo) Update(_ context.Context, t *domainVerification.EmailVerificationToken) error {
	r.updatedEmail = t
	return nil
}
func (*verificationRepo) DeleteExpiredEmailVerificationTokens(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (*verificationRepo) CreatePasswordReset(context.Context, *domainVerification.PasswordResetToken) error {
	return nil
}
func (r *verificationRepo) FindActivePasswordResetByTokenHash(context.Context, string, time.Time) (*domainVerification.PasswordResetToken, error) {
	if r.resetToken == nil {
		return nil, shared.ErrNotFound
	}
	return r.resetToken, nil
}
func (*verificationRepo) InvalidatePasswordResetByUserID(context.Context, uint, time.Time) error {
	return nil
}
func (r *verificationRepo) UpdatePasswordReset(_ context.Context, t *domainVerification.PasswordResetToken) error {
	r.updatedReset = t
	return nil
}
func (*verificationRepo) DeleteExpiredPasswordResetTokens(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type verificationCredentialRepo struct {
	credential *domainAuth.UserCredential
	updated    *domainAuth.UserCredential
}

func (*verificationCredentialRepo) Create(context.Context, *domainAuth.UserCredential) error {
	return nil
}
func (r *verificationCredentialRepo) FindByUserID(context.Context, uint) (*domainAuth.UserCredential, error) {
	if r.credential == nil {
		return nil, shared.ErrNotFound
	}
	return r.credential, nil
}
func (r *verificationCredentialRepo) Update(_ context.Context, c *domainAuth.UserCredential) error {
	r.updated = c
	return nil
}

type verificationHasher struct{}

func (*verificationHasher) Hash(plain string) (string, error) { return "hashed:" + plain, nil }
func (*verificationHasher) Verify(string, string) bool        { return true }

type verificationSessionStore struct{ deletedUserID uint }

func (*verificationSessionStore) Save(context.Context, *domainAuth.RefreshSession) error { return nil }
func (*verificationSessionStore) FindByID(context.Context, string) (*domainAuth.RefreshSession, error) {
	return nil, shared.ErrNotFound
}
func (*verificationSessionStore) FindByUserID(context.Context, uint) (*domainAuth.RefreshSession, error) {
	return nil, shared.ErrNotFound
}
func (*verificationSessionStore) DeleteByID(context.Context, string) error { return nil }
func (s *verificationSessionStore) DeleteByUserID(_ context.Context, id uint) error {
	s.deletedUserID = id
	return nil
}

type verificationBus struct{ events []shared.Event }

func (b *verificationBus) Publish(_ context.Context, events ...shared.Event) error {
	b.events = append(b.events, events...)
	return nil
}
