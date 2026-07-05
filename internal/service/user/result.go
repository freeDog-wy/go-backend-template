package user

import domainUser "github.com/freeDog-wy/go-backend-template/internal/domain/user"

// UserResult 用户返回结果。
type UserResult struct {
	ID    uint
	Name  string
	Email string
}

// FromEntity 从领域实体构建 UserResult。
func FromEntity(e *domainUser.User) *UserResult {
	return &UserResult{
		ID:    e.GetID(),
		Name:  e.GetName(),
		Email: e.GetEmail(),
	}
}
