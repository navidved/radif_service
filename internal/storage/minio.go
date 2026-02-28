package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioStorage implements Storage using a MinIO (or any S3-compatible) backend.
// To switch to ArvanCloud Object Storage, change STORAGE_ENDPOINT and credentials —
// no code changes are needed since ArvanCloud is S3-compatible.
type MinioStorage struct {
	client     *minio.Client
	bucket     string
	publicBase string
}

// NewMinioStorage creates a MinIO client, ensures the bucket exists with a public-read
// policy, and returns a ready-to-use MinioStorage.
func NewMinioStorage(endpoint, accessKey, secretKey, bucket, publicBase string, useSSL bool) (*MinioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	ctx := context.Background()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket existence: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket %q: %w", bucket, err)
		}
		log.Printf("storage: created bucket %q", bucket)
	}

	if err := client.SetBucketPolicy(ctx, bucket, publicReadPolicy(bucket)); err != nil {
		return nil, fmt.Errorf("set bucket policy: %w", err)
	}

	return &MinioStorage{
		client:     client,
		bucket:     bucket,
		publicBase: strings.TrimRight(publicBase, "/"),
	}, nil
}

// Upload streams reader to MinIO under key. size must be the exact byte count
// (pass -1 only if the size is genuinely unknown — MinIO will buffer it).
func (s *MinioStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

// Delete removes the object at key from the bucket.
func (s *MinioStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// PublicURL returns the browser-accessible URL for the given key.
// For local MinIO: "http://localhost:9000/avatars/user-id/file.jpg"
// For ArvanCloud CDN: "https://cdn.radif.ir/user-id/file.jpg"
func (s *MinioStorage) PublicURL(key string) string {
	return s.publicBase + "/" + key
}

// publicReadPolicy returns an S3 bucket policy JSON that allows anonymous GET on all objects.
func publicReadPolicy(bucket string) string {
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect":    "Allow",
				"Principal": "*",
				"Action":    "s3:GetObject",
				"Resource":  fmt.Sprintf("arn:aws:s3:::%s/*", bucket),
			},
		},
	}
	b, _ := json.Marshal(policy)
	return string(b)
}
