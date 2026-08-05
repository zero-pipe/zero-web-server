package objectstore

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"zero-web-server/internal/port"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIO S3 兼容对象存储（MinIO / 兼容网关）。
type MinIO struct {
	cfg    port.ObjectStoreConfig
	client *minio.Client
	bucket string
}

func NewMinIO(cfg port.ObjectStoreConfig) (*MinIO, error) {
	client, bucket, err := newS3CompatibleClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("minio: %w", err)
	}
	return &MinIO{cfg: cfg, client: client, bucket: bucket}, nil
}

func (m *MinIO) Provider() string { return "minio" }

func (m *MinIO) Health(ctx context.Context) error {
	ok, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("minio: bucket %s not found", m.bucket)
	}
	return nil
}

func (m *MinIO) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}
	_, err := m.client.PutObject(ctx, m.bucket, strings.TrimLeft(key, "/"), r, size, opts)
	return err
}

func (m *MinIO) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, strings.TrimLeft(key, "/"), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (m *MinIO) Delete(ctx context.Context, key string) error {
	return m.client.RemoveObject(ctx, m.bucket, strings.TrimLeft(key, "/"), minio.RemoveObjectOptions{})
}

func (m *MinIO) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if m.cfg.PublicBase != "" {
		return strings.TrimRight(m.cfg.PublicBase, "/") + "/" + strings.TrimLeft(key, "/"), nil
	}
	if expiry <= 0 {
		expiry = time.Hour
	}
	u, err := m.client.PresignedGetObject(ctx, m.bucket, strings.TrimLeft(key, "/"), expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// S3 官方/兼容端点，复用 minio-go（SigV4）。
type S3 struct {
	cfg    port.ObjectStoreConfig
	client *minio.Client
	bucket string
}

func NewS3(cfg port.ObjectStoreConfig) (*S3, error) {
	client, bucket, err := newS3CompatibleClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("s3: %w", err)
	}
	return &S3{cfg: cfg, client: client, bucket: bucket}, nil
}

func (s *S3) Provider() string { return "s3" }

func (s *S3) Health(ctx context.Context) error {
	ok, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("s3: bucket %s not found", s.bucket)
	}
	return nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}
	_, err := s.client.PutObject(ctx, s.bucket, strings.TrimLeft(key, "/"), r, size, opts)
	return err
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, strings.TrimLeft(key, "/"), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, strings.TrimLeft(key, "/"), minio.RemoveObjectOptions{})
}

func (s *S3) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if s.cfg.PublicBase != "" {
		return strings.TrimRight(s.cfg.PublicBase, "/") + "/" + strings.TrimLeft(key, "/"), nil
	}
	if expiry <= 0 {
		expiry = time.Hour
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, strings.TrimLeft(key, "/"), expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func newS3CompatibleClient(cfg port.ObjectStoreConfig) (*minio.Client, string, error) {
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		return nil, "", fmt.Errorf("bucket required")
	}
	if strings.TrimSpace(cfg.AccessKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, "", fmt.Errorf("accessKey and secretKey required")
	}

	endpoint, secure, err := normalizeEndpoint(cfg.Endpoint, cfg.UseSSL)
	if err != nil {
		return nil, "", err
	}
	// 官方 AWS：未填 endpoint 时用 s3.<region>.amazonaws.com
	if endpoint == "" {
		region := strings.TrimSpace(cfg.Region)
		if region == "" {
			region = "us-east-1"
		}
		endpoint = fmt.Sprintf("s3.%s.amazonaws.com", region)
		secure = true
	}

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: secure,
		Region: strings.TrimSpace(cfg.Region),
	}
	if cfg.PathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}

	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, "", err
	}
	return client, bucket, nil
}

func normalizeEndpoint(raw string, useSSL bool) (host string, secure bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", useSSL, nil
	}
	secure = useSSL
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, perr := url.Parse(raw)
		if perr != nil {
			return "", false, perr
		}
		secure = u.Scheme == "https"
		host = u.Host
		if host == "" {
			return "", false, fmt.Errorf("invalid endpoint")
		}
		return host, secure, nil
	}
	return strings.TrimRight(raw, "/"), secure, nil
}

var _ port.ObjectStore = (*MinIO)(nil)
var _ port.ObjectStore = (*S3)(nil)
