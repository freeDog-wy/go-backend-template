//go:build integration

package captcha

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
)

func TestRedisStoreIntegrationVerifyAndClear(t *testing.T) {
	ctx := context.Background()
	rdb := testsupport.OpenRedis(t)
	seed := time.Now().UnixNano()
	store := NewRedisStore(rdb, fmt.Sprintf("integration:captcha:%d:", seed), time.Minute)
	const id = "captcha-id"
	t.Cleanup(func() { _ = rdb.Del(ctx, store.key(id)).Err() })

	if err := store.Set(id, "AbC123"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	ttl, err := rdb.PTTL(ctx, store.key(id)).Result()
	if err != nil || ttl <= 0 {
		t.Fatalf("captcha TTL = %v, %v", ttl, err)
	}
	if !store.Verify(id, "aBc123", false) {
		t.Fatal("Verify() should be case-insensitive")
	}
	if !store.Verify(id, "ABC123", true) {
		t.Fatal("Verify(clear=true) should succeed")
	}
	if store.Verify(id, "abc123", false) {
		t.Fatal("cleared captcha should not verify")
	}
	if store.Verify("missing", "abc123", false) {
		t.Fatal("missing captcha should not verify")
	}
}
