package audit

import (
	"context"
	"errors"
	"testing"
)

func TestDBRecorderRecordsAuditLog(t *testing.T) {
	t.Parallel()

	actorUserID := uint(42)
	writer := &stubWriter{}
	recorder := NewRecorder(writer)
	input := RecordInput{
		ActorUserID: &actorUserID,
		TargetType:  "article",
		TargetID:    "100",
		Action:      "cms_article_published",
		Result:      ResultSuccess,
		IP:          "127.0.0.1",
		UserAgent:   "test-agent",
		Metadata:    map[string]any{"locale": "zh-CN"},
	}

	if err := recorder.Record(context.Background(), input); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if writer.log == nil {
		t.Fatal("writer did not receive audit log")
	}
	if writer.log.GetActorUserID() != input.ActorUserID ||
		writer.log.GetTargetType() != input.TargetType ||
		writer.log.GetTargetID() != input.TargetID ||
		writer.log.GetAction() != input.Action ||
		writer.log.GetResult() != input.Result ||
		writer.log.GetIP() != input.IP ||
		writer.log.GetUserAgent() != input.UserAgent {
		t.Fatalf("persisted audit log = %#v, want input fields", writer.log)
	}
	if got := writer.log.GetMetadata()["locale"]; got != "zh-CN" {
		t.Fatalf("metadata locale = %v, want zh-CN", got)
	}
}

func TestDBRecorderReturnsWriterError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("database unavailable")
	recorder := NewRecorder(&stubWriter{err: wantErr})
	err := recorder.Record(context.Background(), RecordInput{
		TargetType: "user",
		TargetID:   "42",
		Action:     "login",
		Result:     ResultSuccess,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Record() error = %v, want %v", err, wantErr)
	}
}

func TestDBRecorderRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	err := NewRecorder(&stubWriter{}).Record(context.Background(), RecordInput{
		TargetID: "42",
		Action:   "login",
		Result:   ResultSuccess,
	})
	if !errors.Is(err, ErrInvalidAuditLog) {
		t.Fatalf("Record() error = %v, want %v", err, ErrInvalidAuditLog)
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
