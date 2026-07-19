package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformIdempotency "github.com/freeDog-wy/go-backend-template/internal/platform/idempotency"
	"github.com/gin-gonic/gin"
)

func TestIdempotencyReplaysCompletedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &idempotencyStoreFake{}
	r := newIdempotencyRouter(store)

	first := requestWithKey(r, "same-key", `{"title":"one"}`)
	second := requestWithKey(r, "same-key", `{"title":"one"}`)
	if first.Body.String() != second.Body.String() || store.claims != 2 || store.completes != 1 {
		t.Fatalf("first=%s second=%s claims=%d completes=%d", first.Body.String(), second.Body.String(), store.claims, store.completes)
	}
}

func TestIdempotencyRejectsKeyReusedWithDifferentBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &idempotencyStoreFake{}
	r := newIdempotencyRouter(store)
	_ = requestWithKey(r, "same-key", `{"title":"one"}`)
	response := requestWithKey(r, "same-key", `{"title":"two"}`)
	if !strings.Contains(response.Body.String(), `"IDEMPOTENCY_KEY_REUSED"`) || store.completes != 1 {
		t.Fatalf("response=%s completes=%d", response.Body.String(), store.completes)
	}
}

func TestIdempotencyRejectsDuplicateWhileProcessing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &idempotencyStoreFake{stateAfterClaim: platformIdempotency.StateProcessing}
	r := newIdempotencyRouter(store)
	_ = requestWithKey(r, "same-key", `{"title":"one"}`)
	response := requestWithKey(r, "same-key", `{"title":"one"}`)
	if !strings.Contains(response.Body.String(), `"IDEMPOTENCY_IN_PROGRESS"`) {
		t.Fatalf("response=%s", response.Body.String())
	}
}

func newIdempotencyRouter(store idempotencyStore) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CurrentUserIDKey, uint(7)); c.Next() })
	r.POST("/writes", Idempotency(store), func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"success": true}) })
	return r
}

func requestWithKey(r http.Handler, key, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/writes", strings.NewReader(body))
	req.Header.Set(IdempotencyKeyHeader, key)
	r.ServeHTTP(w, req)
	return w
}

type idempotencyStoreFake struct {
	claim           *platformIdempotency.Claim
	claims          int
	completes       int
	stateAfterClaim platformIdempotency.State
}

func (s *idempotencyStoreFake) Claim(_ context.Context, _ uint, _ string, _ string, _ string, requestHash string) (*platformIdempotency.Claim, error) {
	s.claims++
	if s.claim == nil {
		s.claim = &platformIdempotency.Claim{State: platformIdempotency.StateClaimed, RequestHash: requestHash}
		return s.claim, nil
	}
	if s.claim.RequestHash != requestHash {
		return &platformIdempotency.Claim{State: platformIdempotency.StateMismatch, RequestHash: s.claim.RequestHash}, nil
	}
	if s.stateAfterClaim == platformIdempotency.StateProcessing {
		return &platformIdempotency.Claim{State: platformIdempotency.StateProcessing, RequestHash: requestHash}, nil
	}
	return &platformIdempotency.Claim{State: platformIdempotency.StateCompleted, RequestHash: requestHash, StatusCode: s.claim.StatusCode, ResponseBody: s.claim.ResponseBody}, nil
}

func (s *idempotencyStoreFake) Complete(_ context.Context, claim *platformIdempotency.Claim, body []byte, status int) error {
	s.completes++
	claim.State, claim.ResponseBody, claim.StatusCode = platformIdempotency.StateCompleted, body, status
	return nil
}
