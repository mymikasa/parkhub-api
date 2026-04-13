package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/parkhub/api/internal/domains/sms/domain"
	"github.com/parkhub/api/internal/domains/sms/errs"
	"github.com/redis/go-redis/v9"
)

type RedisSmsCache struct {
	client *redis.Client
}

func NewRedisSmsCache(client *redis.Client) *RedisSmsCache {
	return &RedisSmsCache{client: client}
}

func (c *RedisSmsCache) Store(ctx context.Context, code *domain.SmsCode) error {
	key := smsKey(code.Phone, code.Purpose)
	data, err := json.Marshal(code)
	if err != nil {
		return err
	}
	ttl := time.Until(code.ExpiresAt)
	if ttl <= 0 {
		ttl = time.Minute
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisSmsCache) Retrieve(ctx context.Context, phone string, purpose domain.SmsPurpose) (*domain.SmsCode, error) {
	key := smsKey(phone, purpose)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, errs.ErrCodeNotFound
		}
		return nil, err
	}
	var code domain.SmsCode
	if err := json.Unmarshal(data, &code); err != nil {
		return nil, err
	}
	return &code, nil
}

func (c *RedisSmsCache) MarkUsed(ctx context.Context, phone string, purpose domain.SmsPurpose) error {
	code, err := c.Retrieve(ctx, phone, purpose)
	if err != nil {
		return err
	}
	code.MarkUsed()
	return c.Store(ctx, code)
}

func (c *RedisSmsCache) SetRateLimit(ctx context.Context, phone string, ttl time.Duration) error {
	key := rateLimitKey(phone)
	return c.client.Set(ctx, key, "1", ttl).Err()
}

func (c *RedisSmsCache) CheckRateLimit(ctx context.Context, phone string) (bool, error) {
	key := rateLimitKey(phone)
	val, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func smsKey(phone string, purpose domain.SmsPurpose) string {
	return fmt.Sprintf("sms:code:%s:%s", purpose, phone)
}

func rateLimitKey(phone string) string {
	return fmt.Sprintf("sms:ratelimit:%s", phone)
}
