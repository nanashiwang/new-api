package limiter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestSuccessWindowAtomicReservationAndFailureRelease(t *testing.T) {
	for _, backend := range []string{"memory", "redis"} {
		t.Run(backend, func(t *testing.T) {
			var memory SuccessWindow
			srv := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
			defer client.Close()
			call := func(nonce, mode string) bool {
				if backend == "memory" {
					if mode == "reserve" {
						return memory.Reserve("user", nonce, 2)
					}
					memory.Finish("user", nonce, mode == "success", time.Minute)
					return true
				}
				allowed, err := RedisSuccessWindow(context.Background(), client, "user", nonce, mode, 2, time.Minute, 5*time.Minute)
				if err != nil {
					t.Error(err)
				}
				return allowed
			}
			var wg sync.WaitGroup
			var admitted atomic.Int32
			var ids sync.Map
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					id := fmt.Sprint(i)
					if call(id, "reserve") {
						admitted.Add(1)
						ids.Store(id, true)
					}
				}(i)
			}
			wg.Wait()
			require.EqualValues(t, 2, admitted.Load())
			ids.Range(func(key, value any) bool { call(key.(string), "failure"); return true })
			require.True(t, call("a", "reserve"))
			require.True(t, call("b", "reserve"))
			call("a", "success")
			call("b", "failure")
			require.True(t, call("c", "reserve"))
			require.False(t, call("d", "reserve"))
		})
	}
}

func TestRedisSuccessWindowCrashExpiryAndRenewal(t *testing.T) {
	srv := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	defer client.Close()
	call := func(nonce, mode string) bool {
		allowed, err := RedisSuccessWindow(context.Background(), client, "user", nonce, mode, 1, time.Second, 2*time.Second)
		require.NoError(t, err)
		return allowed
	}
	require.True(t, call("one", "reserve"))
	require.True(t, call("one", "renew"))
	require.False(t, call("two", "reserve"))
	// A crashed worker's key expires even if no completion callback runs.
	srv.FastForward(4 * time.Second)
	require.True(t, call("two", "reserve"))
}
