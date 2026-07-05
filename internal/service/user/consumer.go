package user

import (
	"context"
	"fmt"

	domainUser "github.com/freeDog-wy/go-backend-template/internal/domain/user"
)

// OnUserRegistered 消费 user.registered 事件——发送验证邮件。
func (s *Service) OnUserRegistered(ctx context.Context, evt domainUser.Registered) error {
	if s.emailSender == nil {
		return nil
	}

	link := "/verify-email?token=placeholder"
	body := fmt.Sprintf("欢迎注册，请点击以下链接验证您的邮箱：\n%s", link)

	if err := s.emailSender.Send(evt.Email, "邮箱验证", body); err != nil {
		if s.logger != nil {
			s.logger.Error("verification email failed", "email", evt.Email, "error", err)
		}
		return err
	}

	if s.logger != nil {
		s.logger.Info("verification email sent", "email", evt.Email)
	}
	return nil
}
