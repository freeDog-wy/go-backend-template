package email

// Config SMTP 配置。
// SmtpHost 为空时，使用 DevSender（仅打印日志，不真实发送）。
type Config struct {
	SmtpHost     string // SMTP 地址，如 smtp.example.com
	SmtpPort     int    // 端口：465(SSL) / 587(STARTTLS)
	SmtpUser     string // 登录账号
	SmtpPassword string // 登录密码
	FromAddress  string // 发件人地址
}
