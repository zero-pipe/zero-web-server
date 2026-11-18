package port

import "context"

// PingFunc 用于探测基础设施连通性（MySQL / Redis 等）。
type PingFunc func(ctx context.Context) error
