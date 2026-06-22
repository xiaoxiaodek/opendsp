-- KEYS[1]: freq:adgroup:{agID}:{date}:{uid}
-- KEYS[2]: freq:campaign:{cID}:{date}:{uid}
-- KEYS[3]: budget:daily:{agID}:{date}
-- KEYS[4]: budget:daily:{cID}:{date}
-- KEYS[5]: budget:total:{cID}
-- KEYS[6]: budget:excluded:{date}
-- KEYS[7]: freq:excluded:{date}:{uid}
-- KEYS[8]: balance:{advertiser_id}
-- ARGV[1]: adgroup_freq_cap
-- ARGV[2]: campaign_freq_cap
-- ARGV[3]: adgroup_daily_budget
-- ARGV[4]: campaign_daily_budget
-- ARGV[5]: campaign_total_budget
-- ARGV[6]: bid_price
-- ARGV[7]: adgroup_id
-- ARGV[8]: campaign_id

local ag_freq = tonumber(redis.call('INCR', KEYS[1]))
redis.call('EXPIRE', KEYS[1], 172800)
if ARGV[1] ~= '0' and ag_freq > tonumber(ARGV[1]) then
    redis.call('SADD', KEYS[7], ARGV[7])
    return {0, 'adgroup_freq_cap'}
end

local c_freq = tonumber(redis.call('INCR', KEYS[2]))
redis.call('EXPIRE', KEYS[2], 172800)
if ARGV[2] ~= '0' and c_freq > tonumber(ARGV[2]) then
    redis.call('SADD', KEYS[7], ARGV[7])
    return {0, 'campaign_freq_cap'}
end

local ag_daily = tonumber(redis.call('INCRBY', KEYS[3], ARGV[6]))
redis.call('EXPIRE', KEYS[3], 172800)
if ARGV[3] ~= '0' and ag_daily > tonumber(ARGV[3]) then
    redis.call('SADD', KEYS[6], ARGV[7])
    return {0, 'adgroup_daily_budget_exhausted'}
end

local c_daily = tonumber(redis.call('INCRBY', KEYS[4], ARGV[6]))
redis.call('EXPIRE', KEYS[4], 172800)
if ARGV[4] ~= '0' and c_daily > tonumber(ARGV[4]) then
    redis.call('SADD', KEYS[6], ARGV[8])
    return {0, 'campaign_daily_budget_exhausted'}
end

local c_total = tonumber(redis.call('INCRBY', KEYS[5], ARGV[6]))
if ARGV[5] ~= '0' and c_total > tonumber(ARGV[5]) then
    redis.call('SADD', KEYS[6], ARGV[8])
    return {0, 'campaign_total_budget_exhausted'}
end

local balance = tonumber(redis.call('GET', KEYS[8]) or '0')
if balance < tonumber(ARGV[6]) then
    redis.call('SADD', KEYS[6], ARGV[7])
    return {0, 'insufficient_balance'}
end
redis.call('DECRBY', KEYS[8], ARGV[6])

return {1, 'ok'}
