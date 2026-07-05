package user

import svcUser "github.com/freeDog-wy/go-backend-template/internal/service/user"

// UserResponse 用户响应 DTO。
type UserResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// FromResult 从应用层结果构建响应。
func FromResult(r *svcUser.UserResult) *UserResponse {
	return &UserResponse{
		ID:    r.ID,
		Name:  r.Name,
		Email: r.Email,
	}
}
