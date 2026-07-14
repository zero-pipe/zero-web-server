package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	domaindevice "zero-web-server/internal/domain/device"

	goredis "github.com/redis/go-redis/v9"
)

func (c *Client) UpdateDevice(ctx context.Context, device *domaindevice.Device) error {
	data, err := json.Marshal(device)
	if err != nil {
		return err
	}
	return c.rdb.HSet(ctx, DevicePrefix, device.DeviceID, data).Err()
}

func (c *Client) GetDevice(ctx context.Context, deviceID string) (*domaindevice.Device, error) {
	val, err := c.rdb.HGet(ctx, DevicePrefix, deviceID).Result()
	if err != nil {
		return nil, err
	}
	var device domaindevice.Device
	if err := json.Unmarshal([]byte(val), &device); err != nil {
		return nil, err
	}
	return &device, nil
}

func (c *Client) RemoveDevice(ctx context.Context, deviceID string) error {
	return c.rdb.HDel(ctx, DevicePrefix, deviceID).Err()
}

func (c *Client) PushKeepalive(ctx context.Context, deviceID string, ts int64) error {
	return c.pushDeviceTimestamp(ctx, DeviceKeepalivePrefix+deviceID, ts, keepaliveTTL)
}

func (c *Client) PushRegister(ctx context.Context, deviceID string, ts int64) error {
	return c.pushDeviceTimestamp(ctx, fmt.Sprintf("ZWS_DEVICE_REGISTER:%s", deviceID), ts, registerTTL)
}

func (c *Client) pushDeviceTimestamp(ctx context.Context, key string, ts int64, ttl time.Duration) error {
	if err := c.rdb.RPush(ctx, key, ts).Err(); err != nil {
		return err
	}
	_ = c.rdb.LTrim(ctx, key, -100, -1).Err()
	return c.rdb.Expire(ctx, key, ttl).Err()
}

func (c *Client) GetKeepaliveTimeStamps(ctx context.Context, deviceID string, count int) ([]int64, error) {
	return c.getDeviceTimeStamps(ctx, DeviceKeepalivePrefix+deviceID, count)
}

func (c *Client) GetRegisterTimeStamps(ctx context.Context, deviceID string, count int) ([]int64, error) {
	return c.getDeviceTimeStamps(ctx, fmt.Sprintf("ZWS_DEVICE_REGISTER:%s", deviceID), count)
}

func (c *Client) getDeviceTimeStamps(ctx context.Context, key string, count int) ([]int64, error) {
	if count <= 0 {
		count = 20
	}
	vals, err := c.rdb.LRange(ctx, key, int64(-count-1), -1).Result()
	if err == goredis.Nil {
		return []int64{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(vals))
	for _, v := range vals {
		ts, parseErr := strconv.ParseInt(v, 10, 64)
		if parseErr != nil {
			continue
		}
		if ts < 1_000_000_000_000 {
			ts *= 1000
		}
		out = append(out, ts)
	}
	return out, nil
}

const (
	keepaliveTTL = time.Hour
	registerTTL  = 3 * time.Hour
)
