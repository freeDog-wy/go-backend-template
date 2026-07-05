package email

// Sender 发送邮件接口，与具体业务解耦。
type Sender interface {
	Send(to, subject, body string) error
}

// New 根据配置创建 Sender。
// host 为空返回 DevSender；否则返回 SMTPSender。
func New(cfg Config) Sender {
	if cfg.SmtpHost == "" {
		return &DevSender{}
	}
	return &SMTPSender{
		host: cfg.SmtpHost,
		port: cfg.SmtpPort,
		user: cfg.SmtpUser,
		pass: cfg.SmtpPassword,
		from: cfg.FromAddress,
	}
}
