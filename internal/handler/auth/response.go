package auth

import svcAuth "github.com/freeDog-wy/go-backend-template/internal/service/auth"

// UserResponse 用户响应 DTO。
type UserResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// FromResult 从应用层结果构建响应。
func FromResult(r *svcAuth.UserResult) *UserResponse {
	return &UserResponse{
		ID:    r.ID,
		Name:  r.Name,
		Email: r.Email,
	}
}

type MessageResponse struct {
	Message string `json:"message"`
}
