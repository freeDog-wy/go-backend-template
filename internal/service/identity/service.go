package identity

import (
	"context"
	"errors"

	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
	"github.com/freeDog-wy/go-backend-template/pkg/captcha"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
)

// EmailVerificationIssuer 为 identity 注册流程提供邮箱验证签发能力。
type EmailVerificationIssuer interface {
	IssueEmailVerification(ctx context.Context, user *domainIdentity.User) (domainVerification.EmailVerificationRequested, error)
}

type Service struct {
	tx                 shared.TxManager
	userRepo           domainIdentity.Repository
	pwdHasher          shared.PasswordHasher
	captcha            captcha.Generator
	verificationIssuer EmailVerificationIssuer
	logger             logger.Logger
	eventBus           shared.EventBus
}

func New(
	tx shared.TxManager,
	userRepo domainIdentity.Repository,
	pwdHasher shared.PasswordHasher,
	captcha captcha.Generator,
	verificationIssuer EmailVerificationIssuer,
	logger logger.Logger,
	eventBus shared.EventBus,
) *Service {
	return &Service{
		tx:                 tx,
		userRepo:           userRepo,
		pwdHasher:          pwdHasher,
		captcha:            captcha,
		verificationIssuer: verificationIssuer,
		logger:             logger,
		eventBus:           eventBus,
	}
}

// Register 用户注册——编排验证码校验、邮箱唯一检查、密码哈希、
// 实体创建、事务持久化、领域事件发布。
func (s *Service) Register(ctx context.Context, cmd RegisterCmd) (*UserResult, error) {
	if !s.captcha.Verify(cmd.CaptchaID, cmd.CaptchaCode) {
		return nil, ErrInvalidCaptcha
	}

	if _, err := s.userRepo.FindByEmail(ctx, cmd.Email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, shared.ErrNotFound) {
		return nil, err
	}

	hashed, err := s.pwdHasher.Hash(cmd.Password)
	if err != nil {
		return nil, err
	}

	user, err := domainIdentity.NewUser(cmd.Name, cmd.Email, hashed)
	if err != nil {
		return nil, err
	}

	var result *UserResult
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}

		verificationEvent, err := s.verificationIssuer.IssueEmailVerification(ctx, user)
		if err != nil {
			return err
		}

		events := append([]shared.Event{}, user.Events()...)
		events = append(events, verificationEvent)
		if err := s.eventBus.Publish(ctx, events...); err != nil {
			return err
		}
		user.ClearEvents()
		result = FromEntity(user)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.logger != nil {
		s.logger.Info("user registered", "user_id", result.ID, "email", cmd.Email)
	}
	return result, nil
}
