package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/parkhub/api/services/identity/internal/repository"
	"github.com/redis/go-redis/v9"
)

var _ repository.RefreshTokenRepo = (*RedisRefreshTokenRepo)(nil)

type RedisRefreshTokenRepo struct {
	client *redis.Client
}

func NewRedisRefreshTokenRepo(client *redis.Client) *RedisRefreshTokenRepo {
	return &RedisRefreshTokenRepo{client: client}
}

func (r *RedisRefreshTokenRepo) Save(ctx context.Context, jti, userID string, ttl time.Duration) error {
	key := refreshKey(jti)
	return r.client.Set(ctx, key, userID, ttl).Err()
}

const consumeScript = `
local val = redis.call('GET', KEYS[1])
if not val then
    return false
end
redis.call('DEL', KEYS[1])
return val
`

var consumeRedisScript = redis.NewScript(consumeScript)

func (r *RedisRefreshTokenRepo) Consume(ctx context.Context, jti string) (string, bool, error) {
	key := refreshKey(jti)
	result, err := consumeRedisScript.Run(ctx, r.client, []string{key}).Result()
	if err != nil {
		if err == redis.Nil {
			return "", false, nil
		}
		return "", false, err
	}
	switch v := result.(type) {
	case string:
		return v, true, nil
	default:
		return "", false, nil
	}
}

func (r *RedisRefreshTokenRepo) Revoke(ctx context.Context, jti string) error {
	key := refreshKey(jti)
	return r.client.Del(ctx, key).Err()
}

func refreshKey(jti string) string {
	return fmt.Sprintf("auth:refresh:%s", jti)
}
