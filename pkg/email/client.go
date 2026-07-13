package email

import (
	"fmt"
	"strings"
)

// Sender 发送邮件接口，与具体业务解耦。
type Sender interface {
	Send(to, subject, body string) error
}

// New creates the explicitly selected sender implementation.
func New(cfg Config) (Sender, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(string(cfg.Mode)))) {
	case ModeLog:
		return &DevSender{}, nil
	case ModeSMTP:
		if strings.TrimSpace(cfg.SmtpHost) == "" || cfg.SmtpPort <= 0 || strings.TrimSpace(cfg.FromAddress) == "" {
			return nil, fmt.Errorf("smtp host, port and from address are required when email.mode=smtp")
		}
		return &SMTPSender{
			host: cfg.SmtpHost,
			port: cfg.SmtpPort,
			user: cfg.SmtpUser,
			pass: cfg.SmtpPassword,
			from: cfg.FromAddress,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported email mode %q", cfg.Mode)
	}
}
