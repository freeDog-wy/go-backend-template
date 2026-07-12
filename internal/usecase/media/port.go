package media

import "context"

// MediaAdminService is the inbound port used by administrative adapters.
type MediaAdminService interface {
	RequestUpload(context.Context, UploadRequest) (*UploadResult, error)
	Complete(context.Context, uint, uint) error
	List(context.Context, int, int) ([]MediaResult, int64, error)
	UpsertTranslation(context.Context, uint, string, string, string) error
}

var _ MediaAdminService = (*Service)(nil)
