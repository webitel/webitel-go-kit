-- this script has side-effects, so it requires replicate commands mode
redis.replicate_commands()

local key = KEYS[1]
local burst = ARGV[1]
local rate = ARGV[2]
local period = ARGV[3]
local cost = tonumber(ARGV[4])

local emission_interval = period / rate
local increment = emission_interval * cost
local burst_offset = emission_interval * burst

-- redis returns time as an array containing two integers: seconds of the epoch
-- time (10 digits) and microseconds (6 digits). for convenience we need to
-- convert them to a floating point number. the resulting number is 16 digits,
-- bordering on the limits of a 64-bit double-precision floating point number.
-- adjust the epoch to be relative to Jan 1, 2017 00:00:00 GMT to avoid floating
-- point problems. this approach is good until "date" is 2,483,228,799 (Wed, 09
-- Sep 2048 01:46:39 GMT), when the adjusted value is 16 digits.
local epoch = 1767225600 -- Thursday, 1 January 2026 at 00:00:00
local date = redis.call("TIME")
date = (date[1] - epoch) + (date[2] / 1000000)

local tat = redis.call("GET", key)

if not tat then
  tat = date
else
  tat = tonumber(tat)
end

tat = math.max(tat, date)

local new_tat = tat + increment
local allow_at = new_tat - burst_offset

local diff = date - allow_at
local remaining = diff / emission_interval

if remaining < 0 then
  local reset_after = tat - date
  local retry_after = diff * -1
  return {
    0, -- allowed
    0, -- remaining
    tostring(retry_after),
    tostring(reset_after),
  }
end

local reset_after = new_tat - date
if reset_after > 0 then
  redis.call("SET", key, new_tat, "EX", math.ceil(reset_after))
end
local retry_after = 0 -- -1
return {
  cost,
  remaining,
  tostring(retry_after),
  tostring(reset_after)
}