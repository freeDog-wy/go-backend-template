package audit

import "time"

type AuditLog struct {
	id          uint
	actorUserID *uint
	targetType  string
	targetID    string
	action      string
	result      string
	ip          string
	userAgent   string
	traceID     string
	metadata    map[string]any
	createdAt   time.Time
}

func NewAuditLog(
	actorUserID *uint,
	targetType string,
	targetID string,
	action string,
	result string,
	ip string,
	userAgent string,
	traceID string,
	metadata map[string]any,
) (*AuditLog, error) {
	if targetType == "" || targetID == "" || action == "" || result == "" {
		return nil, ErrInvalidAuditLog
	}

	return &AuditLog{
		actorUserID: actorUserID,
		targetType:  targetType,
		targetID:    targetID,
		action:      action,
		result:      result,
		ip:          ip,
		userAgent:   userAgent,
		traceID:     traceID,
		metadata:    metadata,
	}, nil
}

func ReconstituteAuditLog(
	id uint,
	actorUserID *uint,
	targetType string,
	targetID string,
	action string,
	result string,
	ip string,
	userAgent string,
	traceID string,
	metadata map[string]any,
	createdAt time.Time,
) *AuditLog {
	return &AuditLog{
		id:          id,
		actorUserID: actorUserID,
		targetType:  targetType,
		targetID:    targetID,
		action:      action,
		result:      result,
		ip:          ip,
		userAgent:   userAgent,
		traceID:     traceID,
		metadata:    metadata,
		createdAt:   createdAt,
	}
}

func (l *AuditLog) GetID() uint                 { return l.id }
func (l *AuditLog) GetActorUserID() *uint       { return l.actorUserID }
func (l *AuditLog) GetTargetType() string       { return l.targetType }
func (l *AuditLog) GetTargetID() string         { return l.targetID }
func (l *AuditLog) GetAction() string           { return l.action }
func (l *AuditLog) GetResult() string           { return l.result }
func (l *AuditLog) GetIP() string               { return l.ip }
func (l *AuditLog) GetUserAgent() string        { return l.userAgent }
func (l *AuditLog) GetTraceID() string          { return l.traceID }
func (l *AuditLog) GetMetadata() map[string]any { return l.metadata }
func (l *AuditLog) GetCreatedAt() time.Time     { return l.createdAt }
