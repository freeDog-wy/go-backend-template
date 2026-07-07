package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
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
	userRepo    domainIdentity.Repository
	pwdHasher   PasswordHasher
	captcha     captcha.Generator
	verifyRepo  domainVerification.Repository
	emailSender email.Sender
	siteBaseURL string
	logger      logger.Logger
	eventBus    shared.EventBus
}

func New(
	tx shared.TxManager,
	userRepo domainIdentity.Repository,
	pwdHasher PasswordHasher,
	captcha captcha.Generator,
	verifyRepo domainVerification.Repository,
	emailSender email.Sender,
	siteBaseURL string,
	logger logger.Logger,
	eventBus shared.EventBus,
) *Service {
	return &Service{
		tx:          tx,
		userRepo:    userRepo,
		pwdHasher:   pwdHasher,
		captcha:     captcha,
		verifyRepo:  verifyRepo,
		emailSender: emailSender,
		siteBaseURL: strings.TrimRight(siteBaseURL, "/"),
		logger:      logger,
		eventBus:    eventBus,
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

		verificationEvent, err := s.issueEmailVerification(ctx, user)
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

	s.logger.Info("user registered", "user_id", result.ID, "email", cmd.Email)
	return result, nil
}

func (s *Service) ResendVerification(ctx context.Context, cmd ResendVerificationCmd) error {
	user, err := s.userRepo.FindByEmail(ctx, cmd.Email)
	if err != nil {
		if errors.Is(err, shared.ErrNotFound) {
			return nil
		}
		return err
	}

	if user.IsEmailVerified() || !user.IsPendingVerification() {
		return nil
	}

	return s.tx.Do(ctx, func(ctx context.Context) error {
		verificationEvent, err := s.issueEmailVerification(ctx, user)
		if err != nil {
			return err
		}
		return s.eventBus.Publish(ctx, verificationEvent)
	})
}

func (s *Service) VerifyEmail(ctx context.Context, cmd VerifyEmailCmd) error {
	now := time.Now()
	tokenHash := hashToken(cmd.Token)

	return s.tx.Do(ctx, func(ctx context.Context) error {
		token, err := s.verifyRepo.FindActiveByTokenHash(ctx, tokenHash, now)
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return ErrInvalidVerificationToken
			}
			return err
		}

		user, err := s.userRepo.FindByID(ctx, token.GetUserID())
		if err != nil {
			if errors.Is(err, shared.ErrNotFound) {
				return ErrInvalidVerificationToken
			}
			return err
		}

		user.VerifyEmail()
		if err := user.Activate(); err != nil {
			return err
		}
		if err := s.userRepo.Update(ctx, user); err != nil {
			return err
		}

		if err := token.Consume(now); err != nil {
			return err
		}
		if err := s.verifyRepo.Update(ctx, token); err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) issueEmailVerification(ctx context.Context, user *domainIdentity.User) (domainVerification.EmailVerificationRequested, error) {
	rawToken, err := generateOpaqueToken()
	if err != nil {
		return domainVerification.EmailVerificationRequested{}, err
	}

	now := time.Now()
	if err := s.verifyRepo.InvalidateByUserID(ctx, user.GetID(), now); err != nil {
		return domainVerification.EmailVerificationRequested{}, err
	}

	token, err := domainVerification.NewEmailVerificationToken(
		user.GetID(),
		hashToken(rawToken),
		now.Add(emailVerificationTTL),
	)
	if err != nil {
		return domainVerification.EmailVerificationRequested{}, err
	}
	if err := s.verifyRepo.Create(ctx, token); err != nil {
		return domainVerification.EmailVerificationRequested{}, err
	}

	return domainVerification.EmailVerificationRequested{
		UserID: user.GetID(),
		Email:  user.GetEmail(),
		Token:  rawToken,
	}, nil
}
