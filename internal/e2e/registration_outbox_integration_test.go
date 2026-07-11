//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	domainVerification "github.com/freeDog-wy/go-backend-template/internal/domain/verification"
	"github.com/freeDog-wy/go-backend-template/internal/infra/database"
	infraOutbox "github.com/freeDog-wy/go-backend-template/internal/infra/outbox"
	modelAuth "github.com/freeDog-wy/go-backend-template/internal/model/auth"
	modelIdentity "github.com/freeDog-wy/go-backend-template/internal/model/identity"
	modelOutbox "github.com/freeDog-wy/go-backend-template/internal/model/outbox"
	modelVerification "github.com/freeDog-wy/go-backend-template/internal/model/verification"
	repoAuth "github.com/freeDog-wy/go-backend-template/internal/repository/auth"
	repoIdentity "github.com/freeDog-wy/go-backend-template/internal/repository/identity"
	repoOutbox "github.com/freeDog-wy/go-backend-template/internal/repository/outbox"
	repoVerification "github.com/freeDog-wy/go-backend-template/internal/repository/verification"
	"github.com/freeDog-wy/go-backend-template/internal/testkit"
	identityUsecase "github.com/freeDog-wy/go-backend-template/internal/usecase/identity"
	supportUsecase "github.com/freeDog-wy/go-backend-template/internal/usecase/support"
	verificationUsecase "github.com/freeDog-wy/go-backend-template/internal/usecase/verification"
	"github.com/freeDog-wy/go-backend-template/pkg/email"
)

func TestRegistrationOutboxToVerificationEmail(t *testing.T) {
	db := testkit.OpenPostgres(t)
	if err := db.AutoMigrate(
		&modelIdentity.User{},
		&modelAuth.UserCredential{},
		&modelOutbox.Event{},
		&modelVerification.EmailVerificationToken{},
		&modelVerification.PasswordResetToken{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	userRepo := repoIdentity.New(db)
	credentialRepo := repoAuth.New(db)
	verificationRepo := repoVerification.New(db)
	outboxRepo := repoOutbox.New(db)
	eventBus := infraOutbox.NewEventBus(outboxRepo)
	tx := database.NewTxManager(db)
	hasher := &fakeHasher{}
	verificationSvc := verificationUsecase.New(tx, userRepo, verificationRepo, credentialRepo, hasher, nil, eventBus, nil)
	registrationSvc := identityUsecase.New(tx, userRepo, nil, credentialRepo, hasher, fakeCaptcha{}, verificationSvc, nil, eventBus)

	email := fmt.Sprintf("registration-e2e-%d@example.com", time.Now().UnixNano())
	registered, err := registrationSvc.Register(context.Background(), identityUsecase.RegisterCmd{
		Name: "E2E User", Email: email, Password: "secret123", CaptchaID: "captcha", CaptchaCode: "123456",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	publisher := &capturingPublisher{}
	if err := supportUsecase.NewOutboxPublisher(outboxRepo, publisher, nil, 10).PublishPending(context.Background()); err != nil {
		t.Fatalf("PublishPending() error = %v", err)
	}

	var event domainVerification.EmailVerificationRequested
	for _, message := range publisher.messages {
		if message.eventName == (domainVerification.EmailVerificationRequested{}).EventName() {
			if err := json.Unmarshal(message.payload, &event); err != nil {
				t.Fatalf("unmarshal verification event: %v", err)
			}
			break
		}
	}
	if event.UserID != registered.ID || event.Email != email || event.Token == "" {
		t.Fatalf("verification event = %+v", event)
	}

	sender := &capturingSender{}
	consumer := verificationUsecase.NewConsumer(sender, "https://app.example.test", nil)
	if err := consumer.OnEmailVerificationRequested(context.Background(), event); err != nil {
		t.Fatalf("OnEmailVerificationRequested() error = %v", err)
	}
	if sender.to != email || sender.subject != "邮箱验证" || !strings.Contains(sender.body, "https://app.example.test/verify-email?token="+event.Token) {
		t.Fatalf("email = to:%q subject:%q body:%q", sender.to, sender.subject, sender.body)
	}
}

type fakeCaptcha struct{}

func (fakeCaptcha) Generate() (string, string, error) { return "", "", nil }
func (fakeCaptcha) Verify(string, string) bool        { return true }

type fakeHasher struct{}

func (*fakeHasher) Hash(plain string) (string, error) { return "hash:" + plain, nil }
func (*fakeHasher) Verify(string, string) bool        { return true }

type publishedMessage struct {
	eventName string
	payload   []byte
}
type capturingPublisher struct{ messages []publishedMessage }

func (p *capturingPublisher) Publish(_ context.Context, _ string, eventName string, payload []byte, _ string, _ string) error {
	p.messages = append(p.messages, publishedMessage{eventName: eventName, payload: append([]byte(nil), payload...)})
	return nil
}

type capturingSender struct{ to, subject, body string }

func (s *capturingSender) Send(to, subject, body string) error {
	s.to, s.subject, s.body = to, subject, body
	return nil
}

var _ email.Sender = (*capturingSender)(nil)
