package verification

import (
	"context"
	"errors"
	"time"

	domainIdentity "github.com/freeDog-wy/go-backend-template/internal/domain/identity"
	"github.com/freeDog-wy/go-backend-template/internal/domain/shared"
	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
)

type Service struct {
	tx        shared.TxManager
	userRepo  domainIdentity.Repository
	verifyRepo domainVerification.Repository
	eventBus  shared.EventBus
}

func New(
	tx shared.TxManager,
	userRepo domainIdentity.Repository,
	verifyRepo domainVerification.Repository,
	eventBus shared.EventBus,
) *Service {
	return &Service{
		tx:        tx,
		userRepo:  userRepo,
		verifyRepo: verifyRepo,
		eventBus:  eventBus,
	}
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
		verificationEvent, err := s.IssueEmailVerification(ctx, user)
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

func (s *Service) IssueEmailVerification(ctx context.Context, user *domainIdentity.User) (domainVerification.EmailVerificationRequested, error) {
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
