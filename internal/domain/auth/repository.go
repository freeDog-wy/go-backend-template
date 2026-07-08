package auth

import (
	"context"
	"time"
)

type CredentialRepository interface {
	Create(ctx context.Context, credential *UserCredential) error
	FindByUserID(ctx context.Context, userID uint) (*UserCredential, error)
	Update(ctx context.Context, credential *UserCredential) error
}

type SessionStore interface {
	Save(ctx context.Context, session *RefreshSession) error
	FindByID(ctx context.Context, sessionID string) (*RefreshSession, error)
	FindByUserID(ctx context.Context, userID uint) (*RefreshSession, error)
	DeleteByID(ctx context.Context, sessionID string) error
	DeleteByUserID(ctx context.Context, userID uint) error
}

type AccessTokenManager interface {
	IssueAccessToken(claims AccessClaims) (string, error)
	ParseAccessToken(token string, now time.Time) (*AccessClaims, error)
}
