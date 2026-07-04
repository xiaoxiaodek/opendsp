-- reserve_token.lua
-- Atomically reserve tokens from a sub-bucket for a bid request.
--
-- KEYS[1]: budget:bucket:{campaign_id}:{bucket_id}:tokens
-- KEYS[2]: budget:bucket:{campaign_id}:{bucket_id}:reserved
-- KEYS[3]: budget:reserve:{reservation_id}
-- KEYS[4]: budget:stats:{campaign_id}:{date}
-- KEYS[5]: budget:reserve:index:{adgroup_id}
-- ARGV[1]: request_tokens (cents)
-- ARGV[2]: reservation_id
-- ARGV[3]: campaign_id
-- ARGV[4]: adgroup_id
-- ARGV[5]: bucket_id
-- ARGV[6]: max_reserved_ratio (0.0~1.0)
-- ARGV[7]: ttl_seconds
-- Returns: {status, detail, bucket_id, tokens}

local tokens = tonumber(redis.call('GET', KEYS[1]) or '0')
local reserved = tonumber(redis.call('GET', KEYS[2]) or '0')
local request_tokens = tonumber(ARGV[1])
local max_ratio = tonumber(ARGV[6])

local max_reservable = math.floor(tokens * max_ratio)
local effective_reserved = math.min(reserved, max_reservable)
local available = tokens - effective_reserved

if available < request_tokens then
    return {0, 'bucket_tokens_insufficient', tonumber(ARGV[5]), 0}
end

redis.call('DECRBY', KEYS[1], request_tokens)
redis.call('INCRBY', KEYS[2], request_tokens)

local time_arr = redis.call('TIME')
redis.call('HSET', KEYS[3],
    'campaign_id', ARGV[3],
    'adgroup_id', ARGV[4],
    'bucket_id', ARGV[5],
    'tokens', ARGV[1],
    'created_at', time_arr[1])
redis.call('EXPIRE', KEYS[3], ARGV[7])

redis.call('SADD', KEYS[5], ARGV[2])
redis.call('EXPIRE', KEYS[5], tonumber(ARGV[7]) + 300)

redis.call('HINCRBY', KEYS[4], 'total_reserved', request_tokens)
redis.call('EXPIRE', KEYS[4], 172800)

return {1, ARGV[2], tonumber(ARGV[5]), request_tokens}
