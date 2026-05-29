package rediscache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"go-commerce-api/internal/domain"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr})
}

func NewCache(client *redis.Client, ttl time.Duration) *Cache {
	return &Cache{client: client, ttl: ttl}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) GetItem(ctx context.Context, id string) (domain.Item, bool, error) {
	var item domain.Item
	ok, err := c.get(ctx, itemKey(id), &item)
	return item, ok, err
}

func (c *Cache) SetItem(ctx context.Context, item domain.Item) error {
	return c.set(ctx, itemKey(item.ID), item)
}

func (c *Cache) DeleteItem(ctx context.Context, id string) error {
	return c.client.Del(ctx, itemKey(id)).Err()
}

func (c *Cache) GetOrder(ctx context.Context, id string) (domain.Order, bool, error) {
	var order domain.Order
	ok, err := c.get(ctx, orderKey(id), &order)
	return order, ok, err
}

func (c *Cache) SetOrder(ctx context.Context, order domain.Order) error {
	return c.set(ctx, orderKey(order.ID), order)
}

func (c *Cache) get(ctx context.Context, key string, dest any) (bool, error) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cache) set(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, raw, c.ttl).Err()
}

func itemKey(id string) string {
	return "v1:item:" + id
}

func orderKey(id string) string {
	return "v1:order:" + id
}
