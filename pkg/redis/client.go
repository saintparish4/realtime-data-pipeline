package redis

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"time"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(addr, password string, db int) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		Password: password,
		DB: db,
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

func (c *Client) Close() error {
	return c.rdb.Close()
}