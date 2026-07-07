package verification

// EmailVerificationRequested 请求发送邮箱验证邮件事件。
type EmailVerificationRequested struct {
	UserID uint
	Email  string
	Token  string
}

func (EmailVerificationRequested) EventName() string { return "user.email_verification_requested" }
