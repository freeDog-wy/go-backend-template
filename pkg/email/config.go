package email

type Mode string

const (
	ModeLog  Mode = "log"
	ModeSMTP Mode = "smtp"
)

// Config describes the selected email delivery implementation.
type Config struct {
	Mode         Mode
	SmtpHost     string // SMTP 地址，如 smtp.example.com
	SmtpPort     int    // 端口：465(SSL) / 587(STARTTLS)
	SmtpUser     string // 登录账号
	SmtpPassword string // 登录密码
	FromAddress  string // 发件人地址
}
