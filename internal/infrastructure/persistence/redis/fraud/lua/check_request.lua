-- KEYS[1]: ip_zset
-- KEYS[2]: device_zset
-- KEYS[3]: dyn_ip_zset
-- KEYS[4]: dyn_device_zset
-- ARGV[1]: window_ms
-- ARGV[2]: max_ip_count
-- ARGV[3]: max_device_count
-- ARGV[4]: now_ms
-- ARGV[5]: ip
-- ARGV[6]: device
-- ARGV[7]: request_id
-- Returns: {blocked, ip_count, device_count, ip_blocked, device_blocked, reason}

local now = tonumber(ARGV[4])
local window = tonumber(ARGV[1])
local cutoff = now - window

-- Dynamic blacklist checks (fastest path first)
local dyn_ip = redis.call('ZSCORE', KEYS[3], ARGV[5])
if dyn_ip and string.len(ARGV[5]) > 0 and tonumber(dyn_ip) > now then
    return {1, 0, 0, 1, 0, 'dynamic_blacklist'}
end

local dyn_device = redis.call('ZSCORE', KEYS[4], ARGV[6])
if dyn_device and string.len(ARGV[6]) > 0 and tonumber(dyn_device) > now then
    return {1, 0, 0, 0, 1, 'dynamic_blacklist'}
end

-- IP request rate
redis.call('ZREMRANGEBYSCORE', KEYS[1], 0, cutoff)
redis.call('ZADD', KEYS[1], now, ARGV[7])
local ip_count = redis.call('ZCARD', KEYS[1])

-- Device request rate
redis.call('ZREMRANGEBYSCORE', KEYS[2], 0, cutoff)
redis.call('ZADD', KEYS[2], now, ARGV[7])
local device_count = redis.call('ZCARD', KEYS[2])

local ip_blocked = 0
local device_blocked = 0
local max_ip = tonumber(ARGV[2])
local max_device = tonumber(ARGV[3])

if string.len(ARGV[5]) > 0 and ip_count > max_ip then
    ip_blocked = 1
end
if string.len(ARGV[6]) > 0 and device_count > max_device then
    device_blocked = 1
end

local blocked = 0
if ip_blocked == 1 or device_blocked == 1 then
    blocked = 1
end

return {blocked, ip_count, device_count, ip_blocked, device_blocked, 'request_rate'}
