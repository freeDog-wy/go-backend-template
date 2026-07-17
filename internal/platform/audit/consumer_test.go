package audit

import (
	"context"
	"errors"
	"testing"
)

func TestConsumerOnLogRequestedPersistsAuditLog(t *testing.T) {
	t.Parallel()

	actorUserID := uint(42)
	writer := &stubWriter{}
	consumer := NewConsumer(writer, nil)
	event := LogRequested{
		ActorUserID: &actorUserID,
		TargetType:  "article",
		TargetID:    "100",
		Action:      "cms_article_published",
		Result:      ResultSuccess,
		IP:          "127.0.0.1",
		UserAgent:   "test-agent",
		Metadata:    map[string]any{"locale": "zh-CN"},
	}

	if err := consumer.OnLogRequested(context.Background(), event); err != nil {
		t.Fatalf("OnLogRequested() error = %v", err)
	}
	if writer.log == nil {
		t.Fatal("writer did not receive audit log")
	}
	if writer.log.GetActorUserID() != event.ActorUserID ||
		writer.log.GetTargetType() != event.TargetType ||
		writer.log.GetTargetID() != event.TargetID ||
		writer.log.GetAction() != event.Action ||
		writer.log.GetResult() != event.Result ||
		writer.log.GetIP() != event.IP ||
		writer.log.GetUserAgent() != event.UserAgent {
		t.Fatalf("persisted audit log = %#v, want event fields", writer.log)
	}
	if got := writer.log.GetMetadata()["locale"]; got != "zh-CN" {
		t.Fatalf("metadata locale = %v, want zh-CN", got)
	}
}

func TestConsumerOnLogRequestedReturnsWriterError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("database unavailable")
	consumer := NewConsumer(&stubWriter{err: wantErr}, nil)

	err := consumer.OnLogRequested(context.Background(), LogRequested{
		TargetType: "user",
		TargetID:   "42",
		Action:     "login",
		Result:     ResultSuccess,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("OnLogRequested() error = %v, want %v", err, wantErr)
	}
}

func TestNewAuditLogRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	_, err := NewAuditLog(nil, "", "42", "login", ResultSuccess, "", "", "", nil)
	if !errors.Is(err, ErrInvalidAuditLog) {
		t.Fatalf("NewAuditLog() error = %v, want %v", err, ErrInvalidAuditLog)
	}
}

type stubWriter struct {
	log *AuditLog
	err error
}

func (w *stubWriter) Create(_ context.Context, log *AuditLog) error {
	w.log = log
	return w.err
}
