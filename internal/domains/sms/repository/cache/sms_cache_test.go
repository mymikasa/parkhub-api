package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/parkhub/api/internal/domains/sms/domain"
	"github.com/parkhub/api/internal/domains/sms/errs"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestCache(t *testing.T) (*RedisSmsCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() { client.Close() })
	return NewRedisSmsCache(client), mr
}

func newTestSmsCode(t *testing.T) *domain.SmsCode {
	t.Helper()
	code, err := domain.NewSmsCode("13800138000", domain.SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)
	return code
}

func TestRedisSmsCache_StoreAndRetrieve(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	code := newTestSmsCode(t)
	require.NoError(t, cache.Store(ctx, code))

	retrieved, err := cache.Retrieve(ctx, code.Phone, code.Purpose)
	require.NoError(t, err)
	assert.Equal(t, code.Phone, retrieved.Phone)
	assert.Equal(t, code.Code, retrieved.Code)
	assert.Equal(t, code.Purpose, retrieved.Purpose)
	assert.Equal(t, code.Used, retrieved.Used)
}

func TestRedisSmsCache_Retrieve_NotFound(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	_, err := cache.Retrieve(ctx, "13800138000", domain.SmsPurposeLogin)
	assert.ErrorIs(t, err, errs.ErrCodeNotFound)
}

func TestRedisSmsCache_MarkUsed(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	code := newTestSmsCode(t)
	require.NoError(t, cache.Store(ctx, code))

	require.NoError(t, cache.MarkUsed(ctx, code.Phone, code.Purpose))

	retrieved, err := cache.Retrieve(ctx, code.Phone, code.Purpose)
	require.NoError(t, err)
	assert.True(t, retrieved.Used)
}

func TestRedisSmsCache_MarkUsed_NotFound(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	err := cache.MarkUsed(ctx, "13800138000", domain.SmsPurposeLogin)
	assert.ErrorIs(t, err, errs.ErrCodeNotFound)
}

func TestRedisSmsCache_RateLimit(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	limited, err := cache.CheckRateLimit(ctx, "13800138000")
	require.NoError(t, err)
	assert.False(t, limited)

	require.NoError(t, cache.SetRateLimit(ctx, "13800138000", 60*time.Second))

	limited, err = cache.CheckRateLimit(ctx, "13800138000")
	require.NoError(t, err)
	assert.True(t, limited)
}

func TestRedisSmsCache_RateLimit_Expired(t *testing.T) {
	cache, mr := setupTestCache(t)
	ctx := context.Background()

	require.NoError(t, cache.SetRateLimit(ctx, "13800138000", 5*time.Second))

	limited, err := cache.CheckRateLimit(ctx, "13800138000")
	require.NoError(t, err)
	assert.True(t, limited)

	mr.FastForward(6 * time.Second)

	limited, err = cache.CheckRateLimit(ctx, "13800138000")
	require.NoError(t, err)
	assert.False(t, limited)
}
