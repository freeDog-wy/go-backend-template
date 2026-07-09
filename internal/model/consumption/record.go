package consumption

import (
	"time"

	domainConsumption "github.com/freeDog-wy/go-backend-template/internal/domain/consumption"
)

// Record 是 message_consumptions 表对应的 ORM 模型。
type Record struct {
	ID            uint       `gorm:"primaryKey"`
	ConsumerGroup string     `gorm:"size:191;not null;uniqueIndex:uk_consumer_group_message_key"`
	MessageKey    string     `gorm:"size:191;not null;uniqueIndex:uk_consumer_group_message_key"`
	EventName     string     `gorm:"size:191;index;not null"`
	TraceID       string     `gorm:"size:64;index"`
	Status        string     `gorm:"size:32;index;not null"`
	AttemptCount  int        `gorm:"not null;default:1"`
	LastError     string     `gorm:"type:text"`
	LockedUntil   *time.Time `gorm:"index"`
	ProcessedAt   *time.Time `gorm:"index"`
	CreatedAt     time.Time  `gorm:"index;not null"`
	UpdatedAt     time.Time  `gorm:"index;not null"`
}

func (Record) TableName() string {
	return "message_consumptions"
}

func (r *Record) ToEntity() *domainConsumption.Record {
	return domainConsumption.ReconstituteRecord(
		r.ID,
		r.ConsumerGroup,
		r.MessageKey,
		r.EventName,
		r.TraceID,
		domainConsumption.Status(r.Status),
		r.AttemptCount,
		r.LastError,
		r.LockedUntil,
		r.ProcessedAt,
		r.CreatedAt,
		r.UpdatedAt,
	)
}
