-- KEYS[1]: hash_key
-- ARGV[1]: field
-- ARGV[2]: value
-- ARGV[3]: max_changes
-- ARGV[4]: ttl_ms
-- Returns: {blocked, count}

redis.call('HSET', KEYS[1], ARGV[1], ARGV[2])
redis.call('PEXPIRE', KEYS[1], ARGV[4])
local count = redis.call('HLEN', KEYS[1])

local blocked = 0
local max_changes = tonumber(ARGV[3])
if count > max_changes then
    blocked = 1
end

return {blocked, count}
