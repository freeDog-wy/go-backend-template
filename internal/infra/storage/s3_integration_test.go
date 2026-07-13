//go:build integration

package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/freeDog-wy/go-backend-template/internal/testsupport"
)

func TestS3IntegrationPresignedUploadAndHeadObject(t *testing.T) {
	resource := testsupport.OpenS3(t)
	adapter, err := NewS3(context.Background(), Options{
		Endpoint:        resource.Endpoint,
		Region:          resource.Region,
		AccessKeyID:     resource.AccessKeyID,
		SecretAccessKey: resource.SecretAccessKey,
		Bucket:          resource.Bucket,
		Prefix:          "integration",
		UsePathStyle:    true,
	})
	if err != nil {
		t.Fatalf("new S3 adapter: %v", err)
	}

	key := adapter.ObjectKey("upload.png")
	presigned, err := adapter.PresignUpload(context.Background(), key, "image/png")
	if err != nil {
		t.Fatalf("presign upload: %v", err)
	}
	payload := []byte("test-image-content")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, presigned.URL, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create upload request: %v", err)
	}
	for key, value := range presigned.Headers {
		req.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload via presigned URL: %v", err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatalf("read upload response: %v", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("upload status = %d, want 2xx", response.StatusCode)
	}

	object, err := adapter.HeadObject(context.Background(), key)
	if err != nil {
		t.Fatalf("head object: %v", err)
	}
	if object.Size != int64(len(payload)) {
		t.Fatalf("object size = %d, want %d", object.Size, len(payload))
	}
	if object.ContentType != "image/png" {
		t.Fatalf("object content type = %q, want image/png", object.ContentType)
	}
}
