-- Read-only query for sliding window status.
-- KEYS[1]: limiter key
-- ARGV[1]: window length in milliseconds
-- Returns: {count, oldest_ms, now_ms}

local key = KEYS[1]
local window_ms = tonumber(ARGV[1])

local now = redis.call('TIME')
local now_ms = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local window_start = now_ms - window_ms

redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

local count = redis.call('ZCARD', key)

local oldest_ms = 0
if count > 0 then
    local earliest = redis.call('ZRANGEBYSCORE', key, '-inf', '+inf', 'LIMIT', 0, 1)
    if earliest and #earliest > 0 then
        oldest_ms = tonumber(redis.call('ZSCORE', key, earliest[1])) or 0
    end
end

return {count, oldest_ms, now_ms}
