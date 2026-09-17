// Package storage abstracts S3-compatible object storage (MinIO locally,
// Cloudflare R2 or any S3 provider in production).
package storage

import (
	"context"
	"io"
	"time"
)

// Storage is the interface the rest of the app depends on. Swapping MinIO for
// R2 is a matter of endpoint/credentials, not a code change.
type Storage interface {
	// Put stores an object and returns nothing but an error.
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Get returns a reader for the object; the caller must Close it.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// PresignGet returns a time-limited URL for downloading the object.
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
	// Delete removes the object.
	Delete(ctx context.Context, key string) error
	// EnsureBucket creates the configured bucket if it does not exist.
	EnsureBucket(ctx context.Context) error
	// Ping checks the backend is reachable (used by /healthz).
	Ping(ctx context.Context) error
	// Bucket returns the configured bucket name.
	Bucket() string
}
