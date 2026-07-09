package outbox

import (
	"time"

	domainOutbox "github.com/freeDog-wy/go-backend-template/internal/domain/outbox"
)

// Event 是 outbox_events 表对应的 ORM 模型。
type Event struct {
	ID          uint       `gorm:"primaryKey"`
	EventName   string     `gorm:"size:191;index;not null"`
	Payload     string     `gorm:"type:text;not null"`
	TraceID     string     `gorm:"size:64;index"`
	PublishedAt *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"index;not null"`
}

func (Event) TableName() string {
	return "outbox_events"
}

// ToEntity 将 ORM 模型转换为领域对象。
func (e *Event) ToEntity() *domainOutbox.Event {
	return domainOutbox.ReconstituteEvent(
		e.ID,
		e.EventName,
		e.Payload,
		e.TraceID,
		e.PublishedAt,
		e.CreatedAt,
	)
}

// FromEntity 将领域对象转换为 ORM 模型。
func FromEntity(event *domainOutbox.Event) *Event {
	return &Event{
		ID:          event.GetID(),
		EventName:   event.GetEventName(),
		Payload:     event.GetPayload(),
		TraceID:     event.GetTraceID(),
		PublishedAt: event.GetPublishedAt(),
		CreatedAt:   event.GetCreatedAt(),
	}
}
