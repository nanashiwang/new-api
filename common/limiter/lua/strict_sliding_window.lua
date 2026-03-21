-- Strict sliding window limiter.
-- KEYS[1]: limiter key
-- ARGV[1]: window length in milliseconds
-- ARGV[2]: max requests allowed in the window
-- ARGV[3]: unique request nonce
-- ARGV[4]: key expiration in milliseconds

local key = KEYS[1]
local window_ms = tonumber(ARGV[1])
local max_requests = tonumber(ARGV[2])
local nonce = ARGV[3]
local expire_ms = tonumber(ARGV[4])

local now = redis.call('TIME')
local now_ms = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local window_start = now_ms - window_ms

redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

local count = redis.call('ZCARD', key)
if count >= max_requests then
    redis.call('PEXPIRE', key, expire_ms)
    return 0
end

local member = tostring(now_ms) .. ':' .. nonce
redis.call('ZADD', key, now_ms, member)
redis.call('PEXPIRE', key, expire_ms)

return 1
