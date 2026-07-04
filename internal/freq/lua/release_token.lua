-- release_token.lua
-- Release reserved tokens back to the sub-bucket (bid lost or timeout).
--
-- KEYS[1]: budget:bucket:{campaign_id}:{bucket_id}:tokens
-- KEYS[2]: budget:bucket:{campaign_id}:{bucket_id}:reserved
-- KEYS[3]: budget:reserve:{reservation_id}
-- KEYS[4]: budget:stats:{campaign_id}:{date}
-- KEYS[5]: budget:reserve:index:{adgroup_id}
-- ARGV[1]: reservation_id
-- Returns: {status, message, tokens}

local reserve = redis.call('HGETALL', KEYS[3])
if #reserve == 0 then
    return {0, 'reservation_not_found', 0}
end

local tokens = 0
for i = 1, #reserve, 2 do
    if reserve[i] == 'tokens' then
        tokens = tonumber(reserve[i+1])
        break
    end
end

if tokens <= 0 then
    return {0, 'invalid_tokens', 0}
end

-- Return tokens to bucket
redis.call('INCRBY', KEYS[1], tokens)
redis.call('DECRBY', KEYS[2], tokens)
redis.call('DEL', KEYS[3])
redis.call('SREM', KEYS[5], ARGV[1])
redis.call('HINCRBY', KEYS[4], 'total_released', tokens)

return {1, 'released', tokens}
