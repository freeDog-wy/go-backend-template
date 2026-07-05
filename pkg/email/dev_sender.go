package email

import "fmt"

// DevSender 开发环境发信器——仅打印到控制台，不实际发送。
type DevSender struct{}

func (s *DevSender) Send(to, subject, body string) error {
	fmt.Printf("[DEV EMAIL] To: %s | Subject: %s | Body: %s\n", to, subject, body)
	return nil
}
