package verification

import "time"

type EmailVerificationToken struct {
	id         uint
	userID     uint
	tokenHash  string
	expiresAt  time.Time
	consumedAt *time.Time
	createdAt  time.Time
}

func NewEmailVerificationToken(userID uint, tokenHash string, expiresAt time.Time) (*EmailVerificationToken, error) {
	if userID == 0 || tokenHash == "" || expiresAt.IsZero() {
		return nil, ErrInvalidEmailVerificationToken
	}

	return &EmailVerificationToken{
		userID:    userID,
		tokenHash: tokenHash,
		expiresAt: expiresAt,
	}, nil
}

func ReconstituteEmailVerificationToken(
	id, userID uint,
	tokenHash string,
	expiresAt time.Time,
	consumedAt *time.Time,
	createdAt time.Time,
) *EmailVerificationToken {
	return &EmailVerificationToken{
		id:         id,
		userID:     userID,
		tokenHash:  tokenHash,
		expiresAt:  expiresAt,
		consumedAt: consumedAt,
		createdAt:  createdAt,
	}
}

func (t *EmailVerificationToken) Consume(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidEmailVerificationToken
	}
	t.consumedAt = &at
	return nil
}

func (t *EmailVerificationToken) GetID() uint               { return t.id }
func (t *EmailVerificationToken) GetUserID() uint           { return t.userID }
func (t *EmailVerificationToken) GetTokenHash() string      { return t.tokenHash }
func (t *EmailVerificationToken) GetExpiresAt() time.Time   { return t.expiresAt }
func (t *EmailVerificationToken) GetConsumedAt() *time.Time { return t.consumedAt }
func (t *EmailVerificationToken) GetCreatedAt() time.Time   { return t.createdAt }

type PasswordResetToken struct {
	id         uint
	userID     uint
	tokenHash  string
	expiresAt  time.Time
	consumedAt *time.Time
	createdAt  time.Time
}

func NewPasswordResetToken(userID uint, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error) {
	if userID == 0 || tokenHash == "" || expiresAt.IsZero() {
		return nil, ErrInvalidPasswordResetToken
	}

	return &PasswordResetToken{
		userID:    userID,
		tokenHash: tokenHash,
		expiresAt: expiresAt,
	}, nil
}

func ReconstitutePasswordResetToken(
	id, userID uint,
	tokenHash string,
	expiresAt time.Time,
	consumedAt *time.Time,
	createdAt time.Time,
) *PasswordResetToken {
	return &PasswordResetToken{
		id:         id,
		userID:     userID,
		tokenHash:  tokenHash,
		expiresAt:  expiresAt,
		consumedAt: consumedAt,
		createdAt:  createdAt,
	}
}

func (t *PasswordResetToken) Consume(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidPasswordResetToken
	}
	t.consumedAt = &at
	return nil
}

func (t *PasswordResetToken) GetID() uint               { return t.id }
func (t *PasswordResetToken) GetUserID() uint           { return t.userID }
func (t *PasswordResetToken) GetTokenHash() string      { return t.tokenHash }
func (t *PasswordResetToken) GetExpiresAt() time.Time   { return t.expiresAt }
func (t *PasswordResetToken) GetConsumedAt() *time.Time { return t.consumedAt }
func (t *PasswordResetToken) GetCreatedAt() time.Time   { return t.createdAt }
