//go:build integration

package outbox

import (
	"context"
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
	t.Cleanup(func() {
		_ = db.Where("event_name IN ?", []string{eventOne.GetEventName(), eventTwo.GetEventName()}).Delete(&eventModel{}).Error
	})

	publisher := &recordingPublisher{}
	if err := NewOutboxPublisher(repo, publisher, logger.Noop(), 10).PublishPending(context.Background()); err != nil {
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

type recordingPublisher struct{ events []string }

func (p *recordingPublisher) Publish(_ context.Context, _ string, eventName string, _ []byte, _, _ string) error {
	p.events = append(p.events, eventName)
	return nil
}
