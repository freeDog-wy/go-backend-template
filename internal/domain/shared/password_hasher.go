package shared

// PasswordHasher 密码加密与校验契约。
// 实现在 internal/infra/crypto/。
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(plain, hash string) bool
}
