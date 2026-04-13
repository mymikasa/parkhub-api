package cache

import (
	"context"
	"sync"
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

func TestRedisSmsCache_VerifyAndConsume_Success(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	code := newTestSmsCode(t)
	require.NoError(t, cache.Store(ctx, code))

	err := cache.VerifyAndConsume(ctx, code.Phone, code.Purpose, code.Code)
	assert.NoError(t, err)
}

func TestRedisSmsCache_VerifyAndConsume_ConsumesOnlyOnce(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	code := newTestSmsCode(t)
	require.NoError(t, cache.Store(ctx, code))

	err := cache.VerifyAndConsume(ctx, code.Phone, code.Purpose, code.Code)
	assert.NoError(t, err)

	err = cache.VerifyAndConsume(ctx, code.Phone, code.Purpose, code.Code)
	assert.ErrorIs(t, err, errs.ErrCodeNotFound)
}

func TestRedisSmsCache_VerifyAndConsume_Mismatch(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	code := newTestSmsCode(t)
	require.NoError(t, cache.Store(ctx, code))

	err := cache.VerifyAndConsume(ctx, code.Phone, code.Purpose, "000000")
	assert.ErrorIs(t, err, errs.ErrCodeMismatch)

	err = cache.VerifyAndConsume(ctx, code.Phone, code.Purpose, code.Code)
	assert.ErrorIs(t, err, errs.ErrCodeNotFound)
}

func TestRedisSmsCache_VerifyAndConsume_NotFound(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	err := cache.VerifyAndConsume(ctx, "13800138000", domain.SmsPurposeLogin, "123456")
	assert.ErrorIs(t, err, errs.ErrCodeNotFound)
}

func TestRedisSmsCache_VerifyAndConsume_Expired(t *testing.T) {
	cache, mr := setupTestCache(t)
	ctx := context.Background()

	code := newTestSmsCode(t)
	require.NoError(t, cache.Store(ctx, code))

	mr.FastForward(6 * time.Minute)

	err := cache.VerifyAndConsume(ctx, code.Phone, code.Purpose, code.Code)
	assert.ErrorIs(t, err, errs.ErrCodeNotFound)
}

func TestRedisSmsCache_VerifyAndConsume_ConcurrentOnlyOneWins(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	code := newTestSmsCode(t)
	require.NoError(t, cache.Store(ctx, code))

	const goroutines = 10
	var wg sync.WaitGroup
	successCount := int64(0)
	mu := sync.Mutex{}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cache.VerifyAndConsume(ctx, code.Phone, code.Purpose, code.Code)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), successCount)
}

func TestRedisSmsCache_TryReserveRateLimit_FirstWins(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	reserved, err := cache.TryReserveRateLimit(ctx, "13800138000", 60*time.Second)
	require.NoError(t, err)
	assert.True(t, reserved)
}

func TestRedisSmsCache_TryReserveRateLimit_SecondLoses(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	reserved, err := cache.TryReserveRateLimit(ctx, "13800138000", 60*time.Second)
	require.NoError(t, err)
	assert.True(t, reserved)

	reserved, err = cache.TryReserveRateLimit(ctx, "13800138000", 60*time.Second)
	require.NoError(t, err)
	assert.False(t, reserved)
}

func TestRedisSmsCache_TryReserveRateLimit_Expired(t *testing.T) {
	cache, mr := setupTestCache(t)
	ctx := context.Background()

	reserved, err := cache.TryReserveRateLimit(ctx, "13800138000", 5*time.Second)
	require.NoError(t, err)
	assert.True(t, reserved)

	mr.FastForward(6 * time.Second)

	reserved, err = cache.TryReserveRateLimit(ctx, "13800138000", 60*time.Second)
	require.NoError(t, err)
	assert.True(t, reserved)
}

func TestRedisSmsCache_ReleaseRateLimit(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	reserved, err := cache.TryReserveRateLimit(ctx, "13800138000", 60*time.Second)
	require.NoError(t, err)
	assert.True(t, reserved)

	require.NoError(t, cache.ReleaseRateLimit(ctx, "13800138000"))

	reserved, err = cache.TryReserveRateLimit(ctx, "13800138000", 60*time.Second)
	require.NoError(t, err)
	assert.True(t, reserved)
}

func TestRedisSmsCache_TryReserveRateLimit_ConcurrentOnlyOneWins(t *testing.T) {
	cache, _ := setupTestCache(t)
	ctx := context.Background()

	const goroutines = 10
	var wg sync.WaitGroup
	winCount := int64(0)
	mu := sync.Mutex{}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reserved, err := cache.TryReserveRateLimit(ctx, "13800138000", 60*time.Second)
			if err == nil && reserved {
				mu.Lock()
				winCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), winCount)
}
