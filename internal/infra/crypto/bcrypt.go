// Package crypto 提供密码加密相关的适配器实现。
package crypto

import (
	svcUser "github.com/freeDog-wy/go-backend-template/internal/service/user"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher 基于 bcrypt 实现 service/user.PasswordHasher。
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher 创建 bcrypt 密码加密器。
// cost 为 0 时使用 bcrypt.DefaultCost(10)。
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// 编译期接口检查。
var _ svcUser.PasswordHasher = (*BcryptHasher)(nil)

func (h *BcryptHasher) Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (h *BcryptHasher) Verify(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
