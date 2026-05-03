package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/parkhub/api/services/sms/internal/domain"
	"github.com/parkhub/api/services/sms/internal/errs"
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
	pipe := c.client.Pipeline()
	pipe.Set(ctx, key, data, ttl)
	pipe.Set(ctx, smsTokenKey(code.Phone, code.Purpose), code.Code, ttl)
	_, err = pipe.Exec(ctx)
	return err
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

const verifyAndConsumeScript = `
local token = redis.call('GET', KEYS[1])
if not token then
    return 'not_found'
end
redis.call('DEL', KEYS[1])
if token ~= ARGV[1] then
    return 'mismatch'
end
return 'ok'
`

var verifyScript = redis.NewScript(verifyAndConsumeScript)

func (c *RedisSmsCache) VerifyAndConsume(ctx context.Context, phone string, purpose domain.SmsPurpose, input string) error {
	tokenKey := smsTokenKey(phone, purpose)
	result, err := verifyScript.Run(ctx, c.client, []string{tokenKey}, input).Text()
	if err != nil {
		return err
	}
	switch result {
	case "ok":
		return nil
	case "mismatch":
		return errs.ErrCodeMismatch
	default:
		return errs.ErrCodeNotFound
	}
}

func (c *RedisSmsCache) TryReserveRateLimit(ctx context.Context, phone string, ttl time.Duration) (bool, error) {
	key := rateLimitKey(phone)
	ok, err := c.client.SetNX(ctx, key, "1", ttl).Result()
	return ok, err
}

func (c *RedisSmsCache) ReleaseRateLimit(ctx context.Context, phone string) error {
	key := rateLimitKey(phone)
	return c.client.Del(ctx, key).Err()
}

func smsKey(phone string, purpose domain.SmsPurpose) string {
	return fmt.Sprintf("sms:code:%s:%s", purpose, phone)
}

func smsTokenKey(phone string, purpose domain.SmsPurpose) string {
	return fmt.Sprintf("sms:code:%s:%s:token", purpose, phone)
}

func rateLimitKey(phone string) string {
	return fmt.Sprintf("sms:ratelimit:%s", phone)
}
