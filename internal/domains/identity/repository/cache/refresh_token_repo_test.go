package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRefreshTokenRepo(t *testing.T) (*RedisRefreshTokenRepo, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() { client.Close() })
	return NewRedisRefreshTokenRepo(client), mr
}

func TestRedisRefreshTokenRepo_SaveAndConsume(t *testing.T) {
	repo, _ := setupTestRefreshTokenRepo(t)
	ctx := context.Background()

	err := repo.Save(ctx, "jti-1", "user-1", 5*time.Minute)
	require.NoError(t, err)

	userID, ok, err := repo.Consume(ctx, "jti-1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "user-1", userID)
}

func TestRedisRefreshTokenRepo_Consume_NotFound(t *testing.T) {
	repo, _ := setupTestRefreshTokenRepo(t)
	ctx := context.Background()

	userID, ok, err := repo.Consume(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, userID)
}

func TestRedisRefreshTokenRepo_Consume_OnlyOnce(t *testing.T) {
	repo, _ := setupTestRefreshTokenRepo(t)
	ctx := context.Background()

	err := repo.Save(ctx, "jti-1", "user-1", 5*time.Minute)
	require.NoError(t, err)

	userID, ok, err := repo.Consume(ctx, "jti-1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "user-1", userID)

	userID, ok, err = repo.Consume(ctx, "jti-1")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, userID)
}

func TestRedisRefreshTokenRepo_Consume_Expired(t *testing.T) {
	repo, mr := setupTestRefreshTokenRepo(t)
	ctx := context.Background()

	err := repo.Save(ctx, "jti-1", "user-1", 5*time.Minute)
	require.NoError(t, err)

	mr.FastForward(6 * time.Minute)

	userID, ok, err := repo.Consume(ctx, "jti-1")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, userID)
}

func TestRedisRefreshTokenRepo_Consume_ConcurrentOnlyOneWins(t *testing.T) {
	repo, _ := setupTestRefreshTokenRepo(t)
	ctx := context.Background()

	err := repo.Save(ctx, "jti-1", "user-1", 5*time.Minute)
	require.NoError(t, err)

	const goroutines = 10
	var wg sync.WaitGroup
	successCount := int64(0)
	mu := sync.Mutex{}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok, err := repo.Consume(ctx, "jti-1")
			if err == nil && ok {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), successCount)
}

func TestRedisRefreshTokenRepo_Revoke(t *testing.T) {
	repo, _ := setupTestRefreshTokenRepo(t)
	ctx := context.Background()

	err := repo.Save(ctx, "jti-1", "user-1", 5*time.Minute)
	require.NoError(t, err)

	err = repo.Revoke(ctx, "jti-1")
	require.NoError(t, err)

	userID, ok, err := repo.Consume(ctx, "jti-1")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, userID)
}

func TestRedisRefreshTokenRepo_Revoke_Idempotent(t *testing.T) {
	repo, _ := setupTestRefreshTokenRepo(t)
	ctx := context.Background()

	err := repo.Revoke(ctx, "nonexistent")
	assert.NoError(t, err)
}
