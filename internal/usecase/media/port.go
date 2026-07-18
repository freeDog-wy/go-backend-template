package media

import (
	"context"
	"time"

	domainMedia "github.com/freeDog-wy/go-backend-template/internal/domain/media"
	model "github.com/freeDog-wy/go-backend-template/internal/model/media"
)

// AssetRepository is the persistence capability required by the media use
// cases. Concrete PostgreSQL details remain in internal/repository/media.
type AssetRepository interface {
	Create(ctx context.Context, asset *model.Asset) error
	SetUploadExpiresAt(ctx context.Context, id uint, expiresAt time.Time) error
	Find(ctx context.Context, id uint) (*model.Asset, error)
	MarkReady(ctx context.Context, id uint, mime string, size int64, width, height int, now time.Time) error
	MarkExpired(ctx context.Context, id uint, now time.Time) error
	ClaimCleanupCandidates(ctx context.Context, now, retryBefore time.Time, batchSize int) ([]model.Asset, error)
	MarkDeleted(ctx context.Context, id uint, now time.Time) error
	RecordCleanupFailure(ctx context.Context, id uint, message string) error
	MarkFailed(ctx context.Context, id uint, reason string) error
	List(ctx context.Context, limit, offset int) ([]model.Asset, int64, error)
	ListReadyPublic(ctx context.Context, locale string, ids []uint) ([]domainMedia.PublicAsset, error)
	UpsertTranslation(ctx context.Context, translation *model.Translation) error
}

// MediaAdminService is the inbound port used by administrative adapters.
type MediaAdminService interface {
	RequestUpload(context.Context, UploadRequest) (*UploadResult, error)
	Complete(context.Context, uint, uint) error
	List(context.Context, int, int) ([]MediaResult, int64, error)
	UpsertTranslation(context.Context, uint, string, string, string) error
}

type MediaMaintenanceService interface {
	CleanupStaleUploads(context.Context, int) (int, error)
}

var _ MediaAdminService = (*Service)(nil)
var _ MediaMaintenanceService = (*Service)(nil)
