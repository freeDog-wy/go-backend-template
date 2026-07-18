package audit

import "context"

// Recorder is the platform capability consumed by business use cases. It
// persists audit records using the caller's context and transaction.
type Recorder interface {
	Record(ctx context.Context, input RecordInput) error
}
