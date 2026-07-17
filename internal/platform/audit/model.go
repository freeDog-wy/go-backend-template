package audit

import (
	"encoding/json"
	"time"
)

type logModel struct {
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

func (logModel) TableName() string { return "audit_logs" }

func (l *logModel) toLog() *AuditLog {
	return ReconstituteAuditLog(
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

func logModelFromLog(log *AuditLog) *logModel {
	return &logModel{
		ID:          log.GetID(),
		ActorUserID: log.GetActorUserID(),
		TargetType:  log.GetTargetType(),
		TargetID:    log.GetTargetID(),
		Action:      log.GetAction(),
		Result:      log.GetResult(),
		IP:          log.GetIP(),
		UserAgent:   log.GetUserAgent(),
		TraceID:     log.GetTraceID(),
		Metadata:    encodeMetadata(log.GetMetadata()),
		CreatedAt:   log.GetCreatedAt(),
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
