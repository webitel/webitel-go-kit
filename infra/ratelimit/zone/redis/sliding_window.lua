-- this script has side-effects, so it requires replicate commands mode
redis.replicate_commands()

local epoch = 1767225600 -- Thursday, 1 January 2026 at 00:00:00
local date = redis.call("TIME")
date = (date[1] - epoch) + (date[2] / 1000000)
-- local sec = (now[1] - epoch)
-- local msec = (now[2] / 1000000)
-- now = sec + msec
-- local date = tostring(now)

local key = KEYS[1]
local max = tonumber(ARGV[1]) -- burst:count
local per = math.floor(tonumber(ARGV[2])) -- window:sec
local cost = 1 -- tonumber(ARGV[3])

local start = (date - per) -- when window start[ed]
redis.call('ZREMRANGEBYSCORE', key, 0, start)
local taken = redis.call('ZCARD', key) -- number of token(s) used

local first = redis.call('ZRANGE', key, 0, 0) -- time of the first req within window frame
local last = date -- redis.call('ZRANGE', key, -1, -1) -- time of the previous req within window frame

if not first[1] then
  first = date
  -- last = date
else
  first = tonumber(first[1])
  -- last = tonumber(last[1])
end

local remaining   = ( max - taken - cost )
local reset_after = ( first - start ) -- ( first + per - date )
local retry_after = 0

if remaining < 0 then
  -- DENY --
  retry_after = reset_after
  return {
    0, -- allowed
    0, -- remaining
    tostring(retry_after),
    tostring(reset_after),
  }
end

-- ALLOW --
for i = 0, cost, 1 do
  redis.call('ZADD', key, date, date)
end
redis.call('EXPIRE', key, per)

if remaining < 1 then
  -- the last accepted request in window
  retry_after = reset_after
end

return {
  cost,
  remaining,
  tostring(retry_after),
  tostring(reset_after),
}
