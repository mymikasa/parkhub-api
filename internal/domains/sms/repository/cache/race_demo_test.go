package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestOldCheckThenSet_RateLimit_AllowsBypassUnderConcurrency(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })

	ctx := context.Background()
	key := "sms:ratelimit:13800138000"
	ttl := 60 * time.Second

	var passed int64
	const goroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// old pattern: check then set
			exists, err := client.Exists(ctx, key).Result()
			if err != nil {
				return
			}
			if exists == 0 {
				// simulate network latency between check and set
				time.Sleep(time.Nanosecond)
				err = client.Set(ctx, key, "1", ttl).Err()
				if err != nil {
					return
				}
				atomic.AddInt64(&passed, 1)
			}
		}()
	}
	wg.Wait()

	// old pattern: multiple goroutines can pass the check before any set lands
	require.Greater(t, passed, int64(1), "old check-then-set pattern allows concurrent bypass")
}
