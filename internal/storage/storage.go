// Package storage defines the interface for object storage operations.
// Swap implementations by changing the concrete type injected at startup —
// the MinIO implementation works with any S3-compatible provider (MinIO, ArvanCloud, AWS S3).
package storage

import (
	"context"
	"io"
)

// Storage is the interface for uploading and retrieving objects.
type Storage interface {
	// Upload streams data to the store under the given key.
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	// Delete removes an object identified by key.
	Delete(ctx context.Context, key string) error
	// PublicURL constructs the browser-accessible URL for a given key.
	PublicURL(key string) string
}
