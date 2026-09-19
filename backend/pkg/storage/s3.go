// Package storage mirrors external catalog images into an S3-compatible bucket
// (MinIO) and serves them back, so browsers never hit the (blockable) origin.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/deface90/defshows/backend/pkg/config"
)

// S3 mirrors and serves images from an S3-compatible bucket.
type S3 struct {
	client     *minio.Client
	bucket     string
	publicBase string // <ImagePublicBaseURL>/images
	downloader *http.Client
}

// New builds an S3 storage client. downloader is the HTTP client used to fetch
// source images (routed through the configured proxy); nil falls back to the
// default client.
func New(cfg config.S3, downloader *http.Client) (*S3, error) {
	endpoint := cfg.Endpoint
	secure := strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: new minio client: %w", err)
	}
	if downloader == nil {
		downloader = http.DefaultClient
	}
	base := strings.TrimRight(cfg.ImagePublicBaseURL, "/") + "/images"
	return &S3{client: client, bucket: cfg.Bucket, publicBase: base, downloader: downloader}, nil
}

// PublicURL returns the browser-reachable URL for an object key.
func (s *S3) PublicURL(key string) string {
	return s.publicBase + "/" + strings.TrimLeft(key, "/")
}

// Mirror downloads srcURL and stores it under key (overwriting any existing
// object), returning the public URL of the stored object.
func (s *S3) Mirror(ctx context.Context, srcURL, key string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
	if err != nil {
		return "", fmt.Errorf("storage: mirror request: %w", err)
	}
	resp, err := s.downloader.Do(req)
	if err != nil {
		return "", fmt.Errorf("storage: mirror download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("storage: mirror download: status %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	if _, err := s.client.PutObject(ctx, s.bucket, key, resp.Body, resp.ContentLength, minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		return "", fmt.Errorf("storage: put object: %w", err)
	}
	return s.PublicURL(key), nil
}

// Get returns a reader for the stored object and its content type.
func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("storage: get object: %w", err)
	}
	// GetObject is lazy; Stat surfaces a missing-object error before streaming.
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, "", fmt.Errorf("storage: stat object: %w", err)
	}
	contentType := info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return obj, contentType, nil
}
