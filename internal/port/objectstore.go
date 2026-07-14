package port

import (
	"context"
	"io"
	"time"
)

// ObjectStore 对象存储端口：对接 S3 / MinIO 等，平台自身不实现存储引擎。
// 用于抓拍图、录像文件归档、导出包等「放对象」场景。
type ObjectStore interface {
	// Provider 返回驱动名：noop | minio | s3 | ...
	Provider() string
	// Health 探测连通性（桶是否可达）。
	Health(ctx context.Context) error
	// Put 上传对象。
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Get 下载对象。
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete 删除对象。
	Delete(ctx context.Context, key string) error
	// PresignGet 生成限时下载 URL（浏览器直链）。
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// ObjectStoreConfig 运行时配置视图（与库表对应，供工厂构建 Adapter）。
type ObjectStoreConfig struct {
	Enabled    bool
	Provider   string // noop | minio | s3
	Endpoint   string
	Region     string
	Bucket     string
	AccessKey  string
	SecretKey  string
	UseSSL     bool
	PathStyle  bool // MinIO 常用 path-style
	PublicBase string // 可选：对外访问前缀
}
