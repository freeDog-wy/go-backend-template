package user

// Registered 用户注册成功事件。
type Registered struct {
	UserID uint
	Name   string
	Email  string
}

func (Registered) EventName() string { return "user.registered" }
