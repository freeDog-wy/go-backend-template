package storage

import (
	"context"
	"errors"
	"testing"
)

func TestNewS3ReturnsNotConfiguredForEmptyConfiguration(t *testing.T) {
	t.Parallel()

	_, err := NewS3(context.Background(), Options{})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("NewS3() error = %v, want ErrNotConfigured", err)
	}
}

func TestNewS3RejectsPartialConfiguration(t *testing.T) {
	t.Parallel()

	_, err := NewS3(context.Background(), Options{Endpoint: "http://storage.local"})
	if err == nil || errors.Is(err, ErrNotConfigured) {
		t.Fatalf("NewS3() error = %v, want incomplete configuration error", err)
	}
}
