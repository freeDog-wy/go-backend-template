package idempotency

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultProcessingTTL = time.Minute
	defaultCompletedTTL  = 15 * time.Minute
)

var errClaimExpired = errors.New("idempotency claim is no longer active")

type State string

const (
	StateClaimed    State = "claimed"
	StateProcessing State = "processing"
	StateCompleted  State = "completed"
	StateMismatch   State = "mismatch"
)

// Claim is the short-lived Redis state for one HTTP write request.
type Claim struct {
	State        State
	RequestHash  string
	ownerToken   string
	ResponseBody []byte
	StatusCode   int
	key          string
}

// Store provides short-lived Redis-backed HTTP duplicate protection. It does
// not promise durable, cross-system exactly-once execution.
type Store struct {
	client        *redis.Client
	prefix        string
	processingTTL time.Duration
	completedTTL  time.Duration
}

var claimScript = redis.NewScript(`
local state = redis.call('HGET', KEYS[1], 'state')
if not state then
  redis.call('HSET', KEYS[1], 'state', 'processing', 'request_hash', ARGV[1], 'owner', ARGV[2])
  redis.call('PEXPIRE', KEYS[1], ARGV[3])
  return {'claimed', ARGV[1], ARGV[2], '', ''}
end
local requestHash = redis.call('HGET', KEYS[1], 'request_hash')
if requestHash ~= ARGV[1] then
  return {'mismatch', requestHash or '', '', '', ''}
end
if state == 'completed' then
  return {'completed', requestHash, '', redis.call('HGET', KEYS[1], 'status') or '200', redis.call('HGET', KEYS[1], 'response') or ''}
end
return {'processing', requestHash, '', '', ''}
`)

var completeScript = redis.NewScript(`
if redis.call('HGET', KEYS[1], 'state') ~= 'processing' then return 0 end
if redis.call('HGET', KEYS[1], 'request_hash') ~= ARGV[1] then return 0 end
if redis.call('HGET', KEYS[1], 'owner') ~= ARGV[2] then return 0 end
redis.call('HSET', KEYS[1], 'state', 'completed', 'status', ARGV[3], 'response', ARGV[4])
redis.call('HDEL', KEYS[1], 'owner')
redis.call('PEXPIRE', KEYS[1], ARGV[5])
return 1
`)

func New(client *redis.Client) *Store {
	return NewWithTTL(client, "http_idempotency", defaultProcessingTTL, defaultCompletedTTL)
}

func NewWithTTL(client *redis.Client, prefix string, processingTTL, completedTTL time.Duration) *Store {
	if processingTTL <= 0 {
		processingTTL = defaultProcessingTTL
	}
	if completedTTL <= 0 {
		completedTTL = defaultCompletedTTL
	}
	return &Store{client: client, prefix: strings.Trim(strings.TrimSpace(prefix), ":"), processingTTL: processingTTL, completedTTL: completedTTL}
}

func (s *Store) Claim(ctx context.Context, actorID uint, method, route, key, requestHash string) (*Claim, error) {
	if s == nil || s.client == nil || actorID == 0 || strings.TrimSpace(method) == "" || strings.TrimSpace(route) == "" || strings.TrimSpace(key) == "" || strings.TrimSpace(requestHash) == "" {
		return nil, errors.New("idempotency store is not configured")
	}
	owner, err := randomToken()
	if err != nil {
		return nil, err
	}
	redisKey := s.key(actorID, method, route, key)
	values, err := claimScript.Run(ctx, s.client, []string{redisKey}, requestHash, owner, s.processingTTL.Milliseconds()).StringSlice()
	if err != nil {
		return nil, err
	}
	if len(values) != 5 {
		return nil, fmt.Errorf("unexpected idempotency claim response")
	}
	status, err := strconv.Atoi(values[3])
	if values[3] == "" {
		status = 0
	} else if err != nil {
		return nil, fmt.Errorf("decode idempotency status: %w", err)
	}
	return &Claim{State: State(values[0]), RequestHash: values[1], ownerToken: values[2], StatusCode: status, ResponseBody: []byte(values[4]), key: redisKey}, nil
}

func (s *Store) Complete(ctx context.Context, claim *Claim, body []byte, statusCode int) error {
	if s == nil || s.client == nil || claim == nil || claim.State != StateClaimed || claim.key == "" || claim.ownerToken == "" {
		return errClaimExpired
	}
	completed, err := completeScript.Run(ctx, s.client, []string{claim.key}, claim.RequestHash, claim.ownerToken, statusCode, body, s.completedTTL.Milliseconds()).Int()
	if err != nil {
		return err
	}
	if completed != 1 {
		return errClaimExpired
	}
	return nil
}

func (s *Store) key(actorID uint, method, route, key string) string {
	sum := sha256.Sum256([]byte(strconv.FormatUint(uint64(actorID), 10) + "\x00" + method + "\x00" + route + "\x00" + key))
	prefix := s.prefix
	if prefix == "" {
		prefix = "http_idempotency"
	}
	return prefix + ":" + hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate idempotency owner token: %w", err)
	}
	return hex.EncodeToString(value), nil
}
