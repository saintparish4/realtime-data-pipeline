package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(addr, password string, db int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &Client{rdb: rdb}
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.rdb.Set(ctx, key, data, expiration).Err()
}

func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

func (c *Client) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	data, err := json.Marshal(member)
	if err != nil {
		return err
	}

	return c.rdb.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
}

func (c *Client) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return c.rdb.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

// ZRangeByScore returns members with scores between min and max
func (c *Client) ZRangeByScore(ctx context.Context, key string, min, max float64) ([]redis.Z, error) {
	return c.rdb.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", min),
		Max: fmt.Sprintf("%f", max),
	}).Result()
}

// ZRemRangeByScore removes members with scores between min and max
func (c *Client) ZRemRangeByScore(ctx context.Context, key string, min, max float64) (int64, error) {
	return c.rdb.ZRemRangeByScore(ctx, key, fmt.Sprintf("%f", min), fmt.Sprintf("%f", max)).Result()
}

// Expire sets a timeout on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}
