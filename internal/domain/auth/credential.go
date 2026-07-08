package auth

import "time"

type UserCredential struct {
	userID            uint
	passwordHash      string
	passwordChangedAt time.Time
	createdAt         time.Time
	updatedAt         time.Time
}

func NewUserCredential(userID uint, passwordHash string, now time.Time) (*UserCredential, error) {
	if userID == 0 || passwordHash == "" || now.IsZero() {
		return nil, ErrInvalidCredential
	}

	return &UserCredential{
		userID:            userID,
		passwordHash:      passwordHash,
		passwordChangedAt: now,
		createdAt:         now,
		updatedAt:         now,
	}, nil
}

func ReconstituteUserCredential(
	userID uint,
	passwordHash string,
	passwordChangedAt, createdAt, updatedAt time.Time,
) *UserCredential {
	return &UserCredential{
		userID:            userID,
		passwordHash:      passwordHash,
		passwordChangedAt: passwordChangedAt,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}
}

func (c *UserCredential) ChangePassword(passwordHash string, at time.Time) error {
	if passwordHash == "" || at.IsZero() {
		return ErrInvalidCredential
	}

	c.passwordHash = passwordHash
	c.passwordChangedAt = at
	c.updatedAt = at
	return nil
}

func (c *UserCredential) GetUserID() uint                 { return c.userID }
func (c *UserCredential) GetPasswordHash() string         { return c.passwordHash }
func (c *UserCredential) GetPasswordChangedAt() time.Time { return c.passwordChangedAt }
func (c *UserCredential) GetCreatedAt() time.Time         { return c.createdAt }
func (c *UserCredential) GetUpdatedAt() time.Time         { return c.updatedAt }
