package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const DeviceExpiresPrefix = "ZWS_DEVICE_EXPIRES:"

func (c *Client) SetDeviceExpiry(ctx context.Context, serverID, deviceID string, expireAtMs int64) error {
	key := DeviceExpiresPrefix + serverID
	return c.rdb.ZAdd(ctx, key, redis.Z{Score: float64(expireAtMs), Member: deviceID}).Err()
}

func (c *Client) RemoveDeviceExpiry(ctx context.Context, serverID, deviceID string) error {
	key := DeviceExpiresPrefix + serverID
	return c.rdb.ZRem(ctx, key, deviceID).Err()
}

func (c *Client) ListExpiredDevices(ctx context.Context, serverID string) ([]string, error) {
	key := DeviceExpiresPrefix + serverID
	now := float64(time.Now().UnixMilli())
	return c.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{Min: "0", Max: fmt.Sprintf("%f", now)}).Result()
}

func (c *Client) RemoveExpiredDevices(ctx context.Context, serverID string, deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return nil
	}
	key := DeviceExpiresPrefix + serverID
	members := make([]any, len(deviceIDs))
	for i, id := range deviceIDs {
		members[i] = id
	}
	return c.rdb.ZRem(ctx, key, members...).Err()
}
