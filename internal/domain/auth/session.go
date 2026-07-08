package auth

import "time"

type RefreshSession struct {
	id        string
	userID    uint
	tokenHash string
	expiresAt time.Time
}

func NewRefreshSession(id string, userID uint, tokenHash string, expiresAt time.Time) (*RefreshSession, error) {
	if id == "" || userID == 0 || tokenHash == "" || expiresAt.IsZero() {
		return nil, ErrInvalidSession
	}

	return &RefreshSession{
		id:        id,
		userID:    userID,
		tokenHash: tokenHash,
		expiresAt: expiresAt,
	}, nil
}

func ReconstituteRefreshSession(id string, userID uint, tokenHash string, expiresAt time.Time) *RefreshSession {
	return &RefreshSession{
		id:        id,
		userID:    userID,
		tokenHash: tokenHash,
		expiresAt: expiresAt,
	}
}

func (s *RefreshSession) IsExpired(now time.Time) bool {
	return !s.expiresAt.After(now)
}

func (s *RefreshSession) GetID() string           { return s.id }
func (s *RefreshSession) GetUserID() uint         { return s.userID }
func (s *RefreshSession) GetTokenHash() string    { return s.tokenHash }
func (s *RefreshSession) GetExpiresAt() time.Time { return s.expiresAt }
