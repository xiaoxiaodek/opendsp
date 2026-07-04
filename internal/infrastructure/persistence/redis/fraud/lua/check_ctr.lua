-- KEYS[1]: imp_zset
-- ARGV[1]: window_ms
-- ARGV[2]: max_ctr_pct
-- ARGV[3]: now_ms
-- ARGV[4]: member  (prefixed "imp:" or "click:")
-- Returns: {blocked, imps, clicks, ctr}

local now = tonumber(ARGV[3])
local window = tonumber(ARGV[1])
local cutoff = now - window

redis.call('ZREMRANGEBYSCORE', KEYS[1], 0, cutoff)
redis.call('ZADD', KEYS[1], now, ARGV[4])

local members = redis.call('ZRANGEBYSCORE', KEYS[1], cutoff, now)
local imps = 0
local clicks = 0
for _, m in ipairs(members) do
    if string.sub(m, 1, 4) == 'imp:' then
        imps = imps + 1
    elseif string.sub(m, 1, 6) == 'click:' then
        clicks = clicks + 1
    end
end

local ctr = 0
if imps > 0 then
    ctr = clicks / imps * 100
end

local blocked = 0
local max_ctr = tonumber(ARGV[2])
if ctr > max_ctr then
    blocked = 1
end

return {blocked, imps, clicks, ctr}
