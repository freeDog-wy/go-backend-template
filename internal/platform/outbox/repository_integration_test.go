//go:build integration

package outbox

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
)

func TestRepositoryIntegrationPublishLifecycle(t *testing.T) {
	db := testsupport.OpenPostgres(t)
	if err := db.AutoMigrate(&eventModel{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	repo := New(db)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	eventOneName := "outbox.integration.one." + suffix
	eventTwoName := "outbox.integration.two." + suffix
	eventOne, _ := NewEvent(eventOneName, `{"id":1}`, "trace-1", "")
	eventTwo, _ := NewEvent(eventTwoName, `{"id":2}`, "trace-2", "")
	if err := repo.Create(context.Background(), eventOne, eventTwo); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Where("event_name IN ?", []string{eventOneName, eventTwoName}).Delete(&eventModel{}).Error
	})

	now := time.Now().UTC().Truncate(time.Microsecond)
	claimed, err := repo.ClaimUnpublished(context.Background(), "publisher-one", now, time.Minute, 10000)
	if err != nil {
		t.Fatalf("ClaimUnpublished() error = %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("claimed events = %d, want 2", len(claimed))
	}

	var firstID, secondID uint
	for _, event := range claimed {
		if event.GetEventName() == eventOneName {
			firstID = event.GetID()
		}
		if event.GetEventName() == eventTwoName {
			secondID = event.GetID()
		}
	}
	if firstID == 0 || secondID == 0 || firstID >= secondID {
		t.Fatalf("outbox IDs = %d, %d", firstID, secondID)
	}

	publishedAt := now.Add(time.Second)
	marked, err := repo.MarkPublished(context.Background(), firstID, "publisher-two", publishedAt)
	if err != nil {
		t.Fatalf("MarkPublished() with another claimant error = %v", err)
	}
	if marked {
		t.Fatal("another claimant marked the event as published")
	}
	marked, err = repo.MarkPublished(context.Background(), firstID, "publisher-one", publishedAt)
	if err != nil {
		t.Fatalf("MarkPublished() error = %v", err)
	}
	if !marked {
		t.Fatal("claim owner did not mark the event as published")
	}

	remaining, err := repo.ListUnpublished(context.Background(), 10000)
	if err != nil {
		t.Fatalf("ListUnpublished() after mark error = %v", err)
	}
	for _, event := range remaining {
		if event.GetID() == firstID {
			t.Fatal("published event is still unpublished")
		}
		if event.GetID() == secondID {
			return
		}
	}
	t.Fatal("second event was not returned as unpublished")
}

func TestRepositoryIntegrationConcurrentClaimsDoNotOverlap(t *testing.T) {
	db := testsupport.OpenPostgres(t)
	if err := db.AutoMigrate(&eventModel{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	repo := New(db)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	events := make([]*Event, 0, 4)
	for index := 0; index < 4; index++ {
		event, err := NewEvent(fmt.Sprintf("outbox.concurrent.%s.%d", suffix, index), `{"id":1}`, "", "")
		if err != nil {
			t.Fatalf("NewEvent() error = %v", err)
		}
		events = append(events, event)
	}
	if err := repo.Create(context.Background(), events...); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	type claimResult struct {
		events []*Event
		err    error
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	start := make(chan struct{})
	results := make(chan claimResult, 2)
	var wg sync.WaitGroup
	for _, claimant := range []string{"publisher-one", "publisher-two"} {
		wg.Add(1)
		go func(claimant string) {
			defer wg.Done()
			<-start
			claimed, err := repo.ClaimUnpublished(context.Background(), claimant, now, time.Minute, 2)
			results <- claimResult{events: claimed, err: err}
		}(claimant)
	}
	close(start)
	wg.Wait()
	close(results)

	claimedIDs := make(map[uint]struct{}, 4)
	for result := range results {
		if result.err != nil {
			t.Fatalf("ClaimUnpublished() error = %v", result.err)
		}
		for _, event := range result.events {
			if _, exists := claimedIDs[event.GetID()]; exists {
				t.Fatalf("event %d was claimed more than once", event.GetID())
			}
			claimedIDs[event.GetID()] = struct{}{}
		}
	}
	if len(claimedIDs) != len(events) {
		t.Fatalf("unique claimed events = %d, want %d", len(claimedIDs), len(events))
	}
}

func TestRepositoryIntegrationExpiredLeaseCanBeRecovered(t *testing.T) {
	db := testsupport.OpenPostgres(t)
	if err := db.AutoMigrate(&eventModel{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	repo := New(db)
	event, err := NewEvent("outbox.lease-recovery."+fmt.Sprint(time.Now().UnixNano()), `{"id":1}`, "", "")
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	if err := repo.Create(context.Background(), event); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	firstClaim, err := repo.ClaimUnpublished(context.Background(), "publisher-one", now, time.Minute, 1)
	if err != nil {
		t.Fatalf("first ClaimUnpublished() error = %v", err)
	}
	if len(firstClaim) != 1 {
		t.Fatalf("first claim size = %d, want 1", len(firstClaim))
	}

	secondClaim, err := repo.ClaimUnpublished(context.Background(), "publisher-two", now.Add(time.Minute), time.Minute, 1)
	if err != nil {
		t.Fatalf("second ClaimUnpublished() error = %v", err)
	}
	if len(secondClaim) != 1 || secondClaim[0].GetID() != firstClaim[0].GetID() {
		t.Fatal("expired lease was not recovered by a new claimant")
	}

	marked, err := repo.MarkPublished(context.Background(), firstClaim[0].GetID(), "publisher-one", now.Add(time.Minute+time.Second))
	if err != nil {
		t.Fatalf("old claimant MarkPublished() error = %v", err)
	}
	if marked {
		t.Fatal("old claimant marked an event after its lease was recovered")
	}
	marked, err = repo.MarkPublished(context.Background(), secondClaim[0].GetID(), "publisher-two", now.Add(time.Minute+time.Second))
	if err != nil {
		t.Fatalf("new claimant MarkPublished() error = %v", err)
	}
	if !marked {
		t.Fatal("new claimant did not mark recovered event as published")
	}
}
