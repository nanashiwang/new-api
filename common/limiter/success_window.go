package limiter

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// SuccessWindow counts in-flight reservations plus completed successes.
// Failure releases a reservation without consuming a success-window slot.
type SuccessWindow struct {
	mu          sync.Mutex
	entries     map[string]map[string]time.Time
	lastCleanup time.Time
}

func (w *SuccessWindow) Reserve(key, nonce string, limit int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	if w.entries == nil {
		w.entries = make(map[string]map[string]time.Time)
	}
	if now.Sub(w.lastCleanup) >= time.Minute {
		for key, entries := range w.entries {
			for id, expiry := range entries {
				if !expiry.IsZero() && !expiry.After(now) {
					delete(entries, id)
				}
			}
			if len(entries) == 0 {
				delete(w.entries, key)
			}
		}
		w.lastCleanup = now
	}
	entries := w.entries[key]
	if entries == nil {
		entries = make(map[string]time.Time)
		w.entries[key] = entries
	}
	for id, expiry := range entries {
		if !expiry.IsZero() && !expiry.After(now) {
			delete(entries, id)
		}
	}
	if len(entries) >= limit {
		return false
	}
	entries[nonce] = time.Time{} // live process owns the defer; no timer can evict an active request
	return true
}

func (w *SuccessWindow) Finish(key, nonce string, success bool, window time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	entries := w.entries[key]
	expiry, exists := entries[nonce]
	if !exists || !expiry.IsZero() {
		return
	}
	if success {
		entries[nonce] = time.Now().Add(window)
	} else {
		delete(entries, nonce)
	}
	if len(entries) == 0 {
		delete(w.entries, key)
	}
}

// Use Redis server time and one atomic script for both reservation and outcome.
// Entries are expiration timestamps. Renewable pending leases survive long
// streams but are eventually reclaimed after a crashed process.
var successWindowScript = redis.NewScript(`
local tm = redis.call('TIME')
local now = tm[1] * 1000 + math.floor(tm[2] / 1000)
local pending = 'p:' .. ARGV[2]
local mode = ARGV[1]
local window = tonumber(ARGV[3])
local lease = tonumber(ARGV[4])
if mode == 'reserve' then
  redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now)
  if redis.call('ZCARD', KEYS[1]) >= tonumber(ARGV[5]) then return 0 end
  redis.call('ZADD', KEYS[1], now + lease, pending)
elseif mode == 'renew' then
  if not redis.call('ZSCORE', KEYS[1], pending) then return 0 end
  redis.call('ZADD', KEYS[1], now + lease, pending)
else
  local removed = redis.call('ZREM', KEYS[1], pending)
  if removed == 1 and mode == 'success' then
    redis.call('ZADD', KEYS[1], now + window, 's:' .. ARGV[2])
  end
end
redis.call('PEXPIRE', KEYS[1], math.max(window, lease) + 1000)
return 1
`)

func RedisSuccessWindow(ctx context.Context, client *redis.Client, key, nonce, mode string, limit int, window, lease time.Duration) (bool, error) {
	result, err := successWindowScript.Run(ctx, client, []string{key}, mode, nonce, window.Milliseconds(), lease.Milliseconds(), limit).Int()
	return result == 1, err
}
