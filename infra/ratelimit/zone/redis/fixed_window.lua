-- this script has side-effects, so it requires replicate commands mode
redis.replicate_commands()

local key = KEYS[1] -- limit rate key
local max = tonumber(ARGV[1]) -- max requests per window
local exp = tonumber(ARGV[2]) -- window interval, in seconds
local cost = tonumber(ARGV[3]) -- request(s) count

local reset = exp -- * 1.000 -- msec

-- COMMAND: EXPIRE
-- For instance, incrementing the value of a key with INCR,
-- pushing a new value into a list with LPUSH, or
-- altering the field value of a hash with HSET
-- are all operations that will leave the timeout untouched.
local taken = tonumber(redis.call('INCRBY', key, cost))

if taken == cost then
  -- FIXME: key just been created ? NEW window started !
  redis.call('EXPIRE', key, exp)
  -- redis.call('PEXPIRE', key, exp * 1000)
else
  -- NOTE: each script.call() MUST provide the same 'exp' value !
  -- reset = tonumber(redis.call('TTL', key))
  reset = tonumber(redis.call('PTTL', key)) / 1000 -- msec
end

local remain = ( max - taken )
-- local before = ( taken - cost )
local allow = ( remain < 0 ) and ( cost + remain ) or cost
-- allow = ( allow < 0 ) and 0 or allow

-- if remain < 0 then
--   local allow = ( cost + remain ) --  -remain
-- end

-- if allow < 0 then
--   allow = 0
-- end
-- allow = ( allow < 0 ) and 0 or allow
-- if allow == 0 then
if allow < 0 then
  -- EXCESS ; DENY
  return {
    0, -- allow[ed]
    0, -- remain[ing]
    tostring(reset), -- retry_after
    tostring(reset), -- reset_after
  }
end

if allow < cost then
  -- EXCESS ; ALLOW [partial]
  return {
    allow, -- allow[ed] some ..
    0, -- remain[ing]
    tostring(reset), -- retry_after
    tostring(reset), -- reset_after
  }
end

remain = ( remain < 0 ) and 0 or remain
local retry = ( 0 < remain ) and 0 or reset

return {
  allow, -- allowed some ..
  remain, -- remaining
  tostring(retry), -- retry_after
  tostring(reset), -- reset_after
}
