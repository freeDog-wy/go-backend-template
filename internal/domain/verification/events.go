package verification

// EmailVerificationRequested 请求发送邮箱验证邮件事件。
type EmailVerificationRequested struct {
	UserID uint
	Email  string
	Token  string
}

func (EmailVerificationRequested) EventName() string { return "user.email_verification_requested" }

// PasswordResetRequested 请求发送密码重置邮件事件。
type PasswordResetRequested struct {
	UserID uint
	Email  string
	Token  string
}

func (PasswordResetRequested) EventName() string { return "user.password_reset_requested" }
