package user

import svcUser "github.com/freeDog-wy/go-backend-template/internal/service/user"

// RegisterReq 注册请求 DTO。
type RegisterReq struct {
	Name        string `json:"name"         binding:"required,min=2,max=50"`
	Email       string `json:"email"        binding:"required,email"`
	Password    string `json:"password"     binding:"required,min=6,max=100"`
	CaptchaID   string `json:"captcha_id"   binding:"required"`
	CaptchaCode string `json:"captcha_code" binding:"required"`
}

// ToCommand 转换为应用层命令。
func (r *RegisterReq) ToCommand() svcUser.RegisterCmd {
	return svcUser.RegisterCmd{
		Name:        r.Name,
		Email:       r.Email,
		Password:    r.Password,
		CaptchaID:   r.CaptchaID,
		CaptchaCode: r.CaptchaCode,
	}
}
