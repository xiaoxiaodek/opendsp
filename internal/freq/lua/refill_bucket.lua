-- refill_bucket.lua
-- Refill tokens into a sub-bucket based on pacing rate.
--
-- KEYS[1]: budget:bucket:{campaign_id}:{bucket_id}:tokens
-- KEYS[2]: budget:bucket:{campaign_id}:meta
-- ARGV[1]: refill_amount
-- ARGV[2]: max_tokens
-- ARGV[3]: now_ts (unix timestamp)
-- Returns: {new_tokens, actual_refill}

local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local max_tokens = tonumber(ARGV[2])
local refill = tonumber(ARGV[1])
local new_tokens = math.min(current + refill, max_tokens)
local actual = new_tokens - current

redis.call('SET', KEYS[1], new_tokens)
redis.call('HSET', KEYS[2], 'last_refill_at', ARGV[3])

return {new_tokens, actual}
