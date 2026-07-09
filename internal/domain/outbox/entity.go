package outbox

import (
	"strings"
	"time"
)

// Event 表示落在本地 outbox 表中的待发布事件记录。
type Event struct {
	id          uint
	eventName   string
	payload     string
	traceID     string
	publishedAt *time.Time
	createdAt   time.Time
}

// NewEvent 在事务内创建一条待发布事件，真正投递由后续 publisher 完成。
func NewEvent(eventName, payload, traceID string) (*Event, error) {
	eventName = strings.TrimSpace(eventName)
	if eventName == "" || strings.TrimSpace(payload) == "" {
		return nil, ErrInvalidEvent
	}

	return &Event{
		eventName: eventName,
		payload:   payload,
		traceID:   strings.TrimSpace(traceID),
		createdAt: time.Now(),
	}, nil
}

// ReconstituteEvent 用于从持久化记录恢复领域对象。
func ReconstituteEvent(id uint, eventName, payload, traceID string, publishedAt *time.Time, createdAt time.Time) *Event {
	return &Event{
		id:          id,
		eventName:   eventName,
		payload:     payload,
		traceID:     traceID,
		publishedAt: publishedAt,
		createdAt:   createdAt,
	}
}

// MarkPublished 标记该记录已经成功投递到外部消息系统。
func (e *Event) MarkPublished(now time.Time) {
	e.publishedAt = &now
}

func (e *Event) GetID() uint {
	return e.id
}

func (e *Event) GetEventName() string {
	return e.eventName
}

func (e *Event) GetPayload() string {
	return e.payload
}

func (e *Event) GetTraceID() string {
	return e.traceID
}

func (e *Event) GetPublishedAt() *time.Time {
	return e.publishedAt
}

func (e *Event) GetCreatedAt() time.Time {
	return e.createdAt
}
