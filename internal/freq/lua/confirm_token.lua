-- confirm_token.lua
-- Confirm a reserved token consumption (impression delivered).
--
-- KEYS[1]: budget:bucket:{campaign_id}:{bucket_id}:reserved
-- KEYS[2]: budget:reserve:{reservation_id}
-- KEYS[3]: budget:stats:{campaign_id}:{date}
-- KEYS[4]: budget:reserve:index:{adgroup_id}
-- ARGV[1]: reservation_id
-- Returns: {status, message, tokens}

local reserve = redis.call('HGETALL', KEYS[2])
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

redis.call('DECRBY', KEYS[1], tokens)
redis.call('DEL', KEYS[2])
redis.call('SREM', KEYS[4], ARGV[1])
redis.call('HINCRBY', KEYS[3], 'total_confirmed', tokens)

return {1, 'confirmed', tokens}
