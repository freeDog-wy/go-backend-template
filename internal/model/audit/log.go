package audit

import (
	"encoding/json"
	"time"

	domainAudit "github.com/freeDog-wy/go-backend-template/internal/domain/audit"
)

type Log struct {
	ID          uint      `gorm:"primaryKey"`
	ActorUserID *uint     `gorm:"index"`
	TargetType  string    `gorm:"type:varchar(100);index;not null"`
	TargetID    string    `gorm:"type:varchar(100);index;not null"`
	Action      string    `gorm:"type:varchar(100);index;not null"`
	Result      string    `gorm:"type:varchar(50);index;not null"`
	IP          string    `gorm:"type:varchar(64)"`
	UserAgent   string    `gorm:"type:text"`
	TraceID     string    `gorm:"type:varchar(64);index"`
	Metadata    string    `gorm:"type:jsonb"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (l *Log) ToEntity() *domainAudit.AuditLog {
	return domainAudit.ReconstituteAuditLog(
		l.ID,
		l.ActorUserID,
		l.TargetType,
		l.TargetID,
		l.Action,
		l.Result,
		l.IP,
		l.UserAgent,
		l.TraceID,
		decodeMetadata(l.Metadata),
		l.CreatedAt,
	)
}

func FromEntity(e *domainAudit.AuditLog) *Log {
	return &Log{
		ID:          e.GetID(),
		ActorUserID: e.GetActorUserID(),
		TargetType:  e.GetTargetType(),
		TargetID:    e.GetTargetID(),
		Action:      e.GetAction(),
		Result:      e.GetResult(),
		IP:          e.GetIP(),
		UserAgent:   e.GetUserAgent(),
		TraceID:     e.GetTraceID(),
		Metadata:    encodeMetadata(e.GetMetadata()),
		CreatedAt:   e.GetCreatedAt(),
	}
}

func encodeMetadata(metadata map[string]any) string {
	if len(metadata) == 0 {
		return "{}"
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func decodeMetadata(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return map[string]any{}
	}
	return metadata
}
