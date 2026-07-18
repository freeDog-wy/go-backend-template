//go:build integration

package outbox

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
	"github.com/freeDog-wy/go-backend-template/pkg/logger"
)

func TestOutboxPublisherIntegrationPublishesAndMarksEvents(t *testing.T) {
	db := testsupport.OpenPostgres(t)
	if err := db.AutoMigrate(&eventModel{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	repo := New(db)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	eventOne, _ := NewEvent("outbox.publisher.one."+suffix, `{"id":1}`, "trace-1", "")
	eventTwo, _ := NewEvent("outbox.publisher.two."+suffix, `{"id":2}`, "trace-2", "")
	if err := repo.Create(context.Background(), eventOne, eventTwo); err != nil {
		t.Fatalf("create events: %v", err)
	}

	publisher := &recordingPublisher{}
	if err := NewOutboxPublisher(repo, publisher, logger.Noop(), 10, time.Minute).PublishPending(context.Background()); err != nil {
		t.Fatalf("PublishPending() error = %v", err)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("published events = %d, want 2", len(publisher.events))
	}

	remaining, err := repo.ListUnpublished(context.Background(), 10000)
	if err != nil {
		t.Fatalf("list unpublished: %v", err)
	}
	for _, event := range remaining {
		if event.GetEventName() == eventOne.GetEventName() || event.GetEventName() == eventTwo.GetEventName() {
			t.Fatalf("published event %q is still unpublished", event.GetEventName())
		}
	}
}

func TestOutboxPublisherIntegrationReleasesClaimsAfterPublishFailure(t *testing.T) {
	db := testsupport.OpenPostgres(t)
	if err := db.AutoMigrate(&eventModel{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	repo := New(db)
	event, _ := NewEvent("outbox.publisher.failure."+fmt.Sprint(time.Now().UnixNano()), `{"id":1}`, "", "")
	if err := repo.Create(context.Background(), event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	expectedErr := errors.New("publish failed")
	err := NewOutboxPublisher(repo, failingPublisher{err: expectedErr}, logger.Noop(), 10, time.Minute).PublishPending(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("PublishPending() error = %v, want %v", err, expectedErr)
	}

	claimed, err := repo.ClaimUnpublished(context.Background(), "recovery-publisher", time.Now(), time.Minute, 10)
	if err != nil {
		t.Fatalf("ClaimUnpublished() error = %v", err)
	}
	if len(claimed) != 1 || claimed[0].GetEventName() != event.GetEventName() {
		t.Fatal("failed event claim was not released")
	}
}

type recordingPublisher struct{ events []string }

func (p *recordingPublisher) Publish(_ context.Context, _ string, eventName string, _ []byte, _, _ string) error {
	p.events = append(p.events, eventName)
	return nil
}

type failingPublisher struct{ err error }

func (p failingPublisher) Publish(context.Context, string, string, []byte, string, string) error {
	return p.err
}
