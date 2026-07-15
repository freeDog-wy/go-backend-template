package auth

import (
	"context"
	"testing"
	"time"

	domainServiceAccount "github.com/freeDog-wy/go-backend-template/internal/domain/service_account"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
)

func TestServiceTokenIssuesServiceClaims(t *testing.T) {
	now := time.Now()
	account := domainServiceAccount.ReconstituteServiceAccount(1, 42, "cms-mcp", "hash", "", nil, true, nil, now, now)
	tokenManager := &stubAccessTokenManager{issueToken: "service-token"}
	service := NewServiceTokenService(
		&serviceAccountStub{account: account},
		&stubIdentityRepo{userByID: newTestUser(42, 1, true)},
		&stubSessionStore{},
		&stubPasswordHasher{verifyResult: true},
		tokenManager,
		&stubEventBus{},
		nil,
		"issuer",
		"cms-api",
		10*time.Minute,
	)

	result, err := service.IssueServiceToken(context.Background(), IssueServiceTokenCmd{ClientID: "cms-mcp", ClientSecret: "secret"})
	if err != nil {
		t.Fatalf("IssueServiceToken() error = %v", err)
	}
	if result.AccessToken != "service-token" || result.ExpiresIn != 600 {
		t.Fatalf("IssueServiceToken() result = %+v", result)
	}
	if tokenManager.issuedClaims == nil || tokenManager.issuedClaims.ActorType != "service" || tokenManager.issuedClaims.TokenID == "" {
		t.Fatalf("issued claims = %+v, want service actor and token ID", tokenManager.issuedClaims)
	}
}

func TestServiceTokenRejectsDisabledAccount(t *testing.T) {
	now := time.Now()
	account := domainServiceAccount.ReconstituteServiceAccount(1, 42, "cms-mcp", "hash", "", nil, false, &now, now, now)
	service := NewServiceTokenService(
		&serviceAccountStub{account: account},
		&stubIdentityRepo{userByID: newTestUser(42, 1, true)},
		&stubSessionStore{},
		&stubPasswordHasher{verifyResult: true},
		&stubAccessTokenManager{},
		nil,
		nil,
		"issuer",
		"cms-api",
		10*time.Minute,
	)
	if _, err := service.IssueServiceToken(context.Background(), IssueServiceTokenCmd{ClientID: "cms-mcp", ClientSecret: "secret"}); err != ErrInvalidServiceCredential {
		t.Fatalf("IssueServiceToken() error = %v, want %v", err, ErrInvalidServiceCredential)
	}
}

type serviceAccountStub struct {
	account *domainServiceAccount.ServiceAccount
	err     error
}

func (s *serviceAccountStub) Create(context.Context, *domainServiceAccount.ServiceAccount) error {
	return nil
}
func (s *serviceAccountStub) Update(context.Context, *domainServiceAccount.ServiceAccount) error {
	return nil
}
func (s *serviceAccountStub) FindByClientID(context.Context, string) (*domainServiceAccount.ServiceAccount, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.account, nil
}
func (s *serviceAccountStub) FindByUserID(context.Context, uint) (*domainServiceAccount.ServiceAccount, error) {
	return nil, shared.ErrNotFound
}
