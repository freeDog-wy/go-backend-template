package user

// RegisterCmd 注册命令——与 HTTP 协议无关。
type RegisterCmd struct {
	Name        string
	Email       string
	Password    string
	CaptchaID   string
	CaptchaCode string
}
