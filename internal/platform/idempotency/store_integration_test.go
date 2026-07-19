//go:build integration

package idempotency

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
)

func TestStoreIntegrationClaimCompleteAndReplay(t *testing.T) {
	rdb := testsupport.OpenRedis(t)
	repo := NewWithTTL(rdb, fmt.Sprintf("idempotency_test_%d", time.Now().UnixNano()), time.Minute, time.Minute)
	ctx := context.Background()

	first, err := repo.Claim(ctx, 7, "POST", "/writes", "same-key", "first-hash")
	if err != nil || first.State != StateClaimed {
		t.Fatalf("first Claim() = (%#v, %v), want claimed record", first, err)
	}
	processing, err := repo.Claim(ctx, 7, "POST", "/writes", "same-key", "first-hash")
	if err != nil || processing.State != StateProcessing {
		t.Fatalf("repeat Claim() = (%#v, %v), want processing record", processing, err)
	}
	if err := repo.Complete(ctx, first, []byte(`{"created":true}`), 200); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	completed, err := repo.Claim(ctx, 7, "POST", "/writes", "same-key", "first-hash")
	if err != nil || completed.State != StateCompleted || completed.StatusCode != 200 || !bytes.Equal(completed.ResponseBody, []byte(`{"created":true}`)) {
		t.Fatalf("completed Claim() = (%#v, %v)", completed, err)
	}
	mismatch, err := repo.Claim(ctx, 7, "POST", "/writes", "same-key", "second-hash")
	if err != nil || mismatch.State != StateMismatch {
		t.Fatalf("mismatched Claim() = (%#v, %v)", mismatch, err)
	}
}
