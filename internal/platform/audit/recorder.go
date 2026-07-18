package audit

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// dbRecorder turns a use-case request into a durable audit log. Repository uses
// the transaction stored in ctx when the caller is inside TxManager.Do.
type dbRecorder struct{ repository *Repository }

var _ Recorder = (*dbRecorder)(nil)

func NewRecorder(repository *Repository) Recorder {
	return &dbRecorder{repository: repository}
}

// ResolveRecorder keeps optional recorder injection convenient for tests and
// secondary composition roots. A missing recorder leaves audit disabled.
func ResolveRecorder(recorders ...Recorder) Recorder {
	for _, recorder := range recorders {
		if recorder != nil {
			return recorder
		}
	}
	return nil
}

func (r *dbRecorder) Record(ctx context.Context, input RecordInput) error {
	if r == nil || r.repository == nil {
		return nil
	}
	log, err := NewAuditLog(
		input.ActorUserID,
		input.TargetType,
		input.TargetID,
		input.Action,
		input.Result,
		input.IP,
		input.UserAgent,
		traceIDFromContext(ctx),
		input.Metadata,
	)
	if err != nil {
		return err
	}
	return r.repository.Create(ctx, log)
}

func traceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}
