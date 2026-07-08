package audit

import "context"

type Repository interface {
	Create(ctx context.Context, log *AuditLog) error
}
