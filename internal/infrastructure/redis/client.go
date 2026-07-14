package redis

import (
	"context"
	"fmt"

	"zero-web-server/internal/infrastructure/config"

	goredis "github.com/redis/go-redis/v9"
)

const (
	DevicePrefix           = "ZWS_DEVICE_INFO"
	DeviceKeepalivePrefix  = "ZWS_DEVICE_KEEPALIVE:"
	SIPInviteSessionPrefix = "ZWS_SIP_INVITE_SESSION_INFO:"
	MediaServerPrefix      = "ZWS_MEDIA_SERVER_INFO:"
	ONVIFDevicePrefix      = "ZWS_ONVIF_DEVICE:"
)

type Client struct {
	rdb *goredis.Client
}

func NewClient(cfg config.RedisConfig) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.Database,
		PoolSize: cfg.PoolSize,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

func (c *Client) Raw() *goredis.Client {
	return c.rdb
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) Set(ctx context.Context, key string, value any) error {
	return c.rdb.Set(ctx, key, value, 0).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) HSet(ctx context.Context, key, field string, value any) error {
	return c.rdb.HSet(ctx, key, field, value).Err()
}

func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.rdb.HGet(ctx, key, field).Result()
}

func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.rdb.HDel(ctx, key, fields...).Err()
}
