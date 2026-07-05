package user

import (
	"context"

	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	domainUser "github.com/freeDog-wy/go-backend-template/internal/domain/user"
	"github.com/freeDog-wy/go-backend-template/pkg/captcha"
	"github.com/freeDog-wy/go-backend-template/pkg/email"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
)

// PasswordHasher 密码加密与校验接口——应用层契约。
// 实现在 internal/infra/crypto/。
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Verify(plain, hash string) bool
}

type Service struct {
	tx          shared.TxManager
	userRepo    domainUser.Repository
	pwdHasher   PasswordHasher
	captcha     captcha.Generator
	emailSender email.Sender
	logger      logger.Logger
	eventBus    shared.EventBus
}

func New(
	tx shared.TxManager,
	userRepo domainUser.Repository,
	pwdHasher PasswordHasher,
	captcha captcha.Generator,
	emailSender email.Sender,
	logger logger.Logger,
	eventBus shared.EventBus,
) *Service {
	return &Service{
		tx:          tx,
		userRepo:    userRepo,
		pwdHasher:   pwdHasher,
		captcha:     captcha,
		emailSender: emailSender,
		logger:      logger,
		eventBus:    eventBus,
	}
}

// Register 用户注册——编排验证码校验、邮箱唯一检查、密码哈希、
// 实体创建、事务持久化、领域事件发布。
func (s *Service) Register(ctx context.Context, cmd RegisterCmd) (*UserResult, error) {
	// ① 校验验证码
	if !s.captcha.Verify(cmd.CaptchaID, cmd.CaptchaCode) {
		return nil, ErrInvalidCaptcha
	}

	// ② 邮箱唯一性检查
	if _, err := s.userRepo.FindByEmail(ctx, cmd.Email); err == nil {
		return nil, ErrEmailTaken
	}

	// ③ 密码哈希
	hashed, err := s.pwdHasher.Hash(cmd.Password)
	if err != nil {
		return nil, err
	}

	// ④ 创建领域实体（自动记录 Registered 事件）
	user, err := domainUser.NewUser(cmd.Name, cmd.Email, hashed)
	if err != nil {
		return nil, err
	}

	// ⑤ 事务：持久化 + 发布事件
	var result *UserResult
	err = s.tx.Do(ctx, func(ctx context.Context) error {
		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}
		if err := s.eventBus.Publish(ctx, user.Events()...); err != nil {
			return err
		}
		user.ClearEvents()
		result = FromEntity(user)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// ⑥ 日志
	s.logger.Info("user registered", "user_id", result.ID, "email", cmd.Email)
	return result, nil
}
