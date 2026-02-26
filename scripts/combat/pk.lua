-- combat/pk.lua — PK system formulas
-- All PK-related calculations: lawful penalties, item drops, timers, thresholds.
-- Java reference: L1PinkName.java, L1PcInstance death handler

-- Timer durations (in ticks, 200ms per tick)
local PK_TIMERS = {
    pink_name_ticks = 900,     -- 180 seconds (3 minutes)
    wanted_ticks    = 432000,  -- 24 hours
}

-- PK count thresholds
local PK_THRESHOLDS = {
    warning = 5,   -- start showing red warning at this count
    punish  = 10,  -- referenced in warning message as max before punishment
}

-- get_pk_timers() -> {pink_name_ticks, wanted_ticks}
function get_pk_timers()
    return PK_TIMERS
end

-- get_pk_thresholds() -> {warning, punish}
function get_pk_thresholds()
    return PK_THRESHOLDS
end

-- calc_pk_lawful_penalty(ctx) -> {new_lawful}
-- ctx = {killer_level, killer_lawful}
--
-- Java formula (L1PcInstance):
--   Level < 50:  new_lawful = -(Level^2 * 4)
--   Level >= 50: new_lawful = -(Level^3 * 0.08)
-- Then clamp: if (current_lawful - 1000) < new_lawful, use (current_lawful - 1000)
-- Final clamp to [-32768, 32767]
function calc_pk_lawful_penalty(ctx)
    local level = ctx.killer_level
    local current = ctx.killer_lawful

    local new_lawful
    if level < 50 then
        new_lawful = -1 * math.floor(level * level * 4)
    else
        new_lawful = -1 * math.floor(level * level * level * 0.08)
    end

    -- If already very negative, ensure at least -1000 drop
    if current - 1000 < new_lawful then
        new_lawful = current - 1000
    end

    -- Clamp to int16 range
    if new_lawful < -32768 then
        new_lawful = -32768
    elseif new_lawful > 32767 then
        new_lawful = 32767
    end

    return { new_lawful = new_lawful }
end

-- calc_pk_item_drop(ctx) -> {should_drop, count}
-- ctx = {victim_lawful}
--
-- Java formula:
--   lostRate = ((lawful + 32768) / 1000 - 65) * 4
--   If lostRate >= 0, no drop.
--   Negate lostRate, then double for red-named (lawful < 0).
--   Roll 1-1000: if roll > lostRate, no drop.
--   Drop count based on lawful brackets:
--     <= -30000: 1-4 items
--     <= -20000: 1-3 items
--     <= -10000: 1-2 items
--     else: 1 item
function calc_pk_item_drop(ctx)
    local lawful = ctx.victim_lawful

    -- Blue-named players don't drop items
    if lawful >= 0 then
        return { should_drop = false, count = 0 }
    end

    local lost_rate = math.floor(((lawful + 32768) / 1000 - 65) * 4)
    if lost_rate >= 0 then
        return { should_drop = false, count = 0 }
    end

    lost_rate = -lost_rate

    -- Double rate for red-named
    if lawful < 0 then
        lost_rate = lost_rate * 2
    end

    -- Roll the dice (1-1000)
    local roll = math.random(1, 1000)
    if roll > lost_rate then
        return { should_drop = false, count = 0 }
    end

    -- Determine drop count by lawful bracket
    local count = 1
    if lawful <= -30000 then
        count = math.random(1, 4)
    elseif lawful <= -20000 then
        count = math.random(1, 3)
    elseif lawful <= -10000 then
        count = math.random(1, 2)
    end

    return { should_drop = true, count = count }
end
