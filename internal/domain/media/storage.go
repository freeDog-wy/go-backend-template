package media

import (
	"context"
	"time"
)

// Storage is the outbound port for media object storage.
type Storage interface {
	ObjectKey(name string) string
	PresignUpload(ctx context.Context, key, contentType string) (*PresignedUpload, error)
	HeadObject(ctx context.Context, key string) (*ObjectInfo, error)
}

type PresignedUpload struct {
	URL       string
	Headers   map[string]string
	ExpiresAt time.Time
}

type ObjectInfo struct {
	ContentType string
	Size        int64
}
