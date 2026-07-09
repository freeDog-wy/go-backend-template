package consumption

import "time"

type Status string

const (
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
	StatusDead       Status = "dead"
)

type BeginDecision string

const (
	BeginDecisionAcquired BeginDecision = "acquired"
	BeginDecisionDone     BeginDecision = "done"
	BeginDecisionLocked   BeginDecision = "locked"
)

// Record 表示一条消息在某个 consumer group 下的消费状态。
type Record struct {
	id            uint
	consumerGroup string
	messageKey    string
	eventName     string
	traceID       string
	status        Status
	attemptCount  int
	lastError     string
	lockedUntil   *time.Time
	processedAt   *time.Time
	createdAt     time.Time
	updatedAt     time.Time
}

func ReconstituteRecord(
	id uint,
	consumerGroup, messageKey, eventName, traceID string,
	status Status,
	attemptCount int,
	lastError string,
	lockedUntil, processedAt *time.Time,
	createdAt, updatedAt time.Time,
) *Record {
	return &Record{
		id:            id,
		consumerGroup: consumerGroup,
		messageKey:    messageKey,
		eventName:     eventName,
		traceID:       traceID,
		status:        status,
		attemptCount:  attemptCount,
		lastError:     lastError,
		lockedUntil:   lockedUntil,
		processedAt:   processedAt,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

type BeginCommand struct {
	ConsumerGroup string
	MessageKey    string
	EventName     string
	TraceID       string
	AttemptedAt   time.Time
	LockedUntil   time.Time
}

func (c BeginCommand) Valid() bool {
	return c.ConsumerGroup != "" && c.MessageKey != "" && c.EventName != "" && !c.AttemptedAt.IsZero() && !c.LockedUntil.IsZero()
}

type BeginResult struct {
	Decision     BeginDecision
	AttemptCount int
}

func (r *Record) GetAttemptCount() int {
	return r.attemptCount
}
