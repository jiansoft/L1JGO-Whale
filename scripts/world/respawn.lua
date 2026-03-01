-- Respawn (getback) location definitions
-- Map ID -> {x, y, map}

RESPAWN_LOCATIONS = {
    [0]    = { x = 32583, y = 32929, map = 0 },     -- Silver Knight Village (map 0)
    [4]    = { x = 33084, y = 33391, map = 4 },     -- Silver Knight Village (map 4)
    [2005] = { x = 32689, y = 32842, map = 2005 },  -- Talking Island (starter area)
    [70]   = { x = 32579, y = 32735, map = 70 },    -- Hidden Valley
    [303]  = { x = 32596, y = 32807, map = 303 },   -- Orc Forest
    [350]  = { x = 32657, y = 32857, map = 350 },   -- Elf Forest
}

-- Default: Silver Knight Village (map 4)
local DEFAULT_RESPAWN = { x = 33084, y = 33391, map = 4 }

function get_respawn_location(map_id)
    return RESPAWN_LOCATIONS[map_id] or DEFAULT_RESPAWN
end

-- ======================================================================
-- 回家卷軸目的地（Java: GetbackTable + L1TownLocation）
-- map 4 主大陸依座標找最近城鎮；其他地圖依 getback 映射表
-- ======================================================================

-- 主大陸城鎮座標（Java: L1TownLocation.java 各城鎮中心座標）
local TOWNS = {
    { name = "說話之島",     cx = 32575, cy = 32945, rx = 32575, ry = 32945, map = 0 },
    { name = "銀騎士村莊",   cx = 33084, cy = 33391, rx = 33084, ry = 33391, map = 4 },
    { name = "古魯丁",       cx = 32613, cy = 32775, rx = 32613, ry = 32775, map = 4 },
    { name = "獸人森林",     cx = 32744, cy = 32447, rx = 32744, ry = 32447, map = 4 },
    { name = "風木村莊",     cx = 32620, cy = 33195, rx = 32620, ry = 33195, map = 4 },
    { name = "乘特",         cx = 33050, cy = 32764, rx = 33050, ry = 32764, map = 4 },
    { name = "奇岩",         cx = 33429, cy = 32814, rx = 33429, ry = 32814, map = 4 },
    { name = "海音",         cx = 33600, cy = 33240, rx = 33600, ry = 33240, map = 4 },
    { name = "乘爾登",       cx = 33720, cy = 32500, rx = 33720, ry = 32500, map = 4 },
    { name = "歐瑞",         cx = 34050, cy = 32275, rx = 34050, ry = 32275, map = 4 },
    { name = "妖森",         cx = 33050, cy = 32340, rx = 33050, ry = 32340, map = 4 },
    { name = "亞丁",         cx = 34000, cy = 33140, rx = 34000, ry = 33140, map = 4 },
}

-- 非主大陸地圖→回家目的地（Java: getback 資料表）
local HOME_SCROLL_MAP = {
    -- 說話之島地下城
    [1]    = { x = 32575, y = 32945, map = 0 },   -- 說話之島 1F
    [2]    = { x = 32575, y = 32945, map = 0 },   -- 說話之島 2F
    [3]    = { x = 32575, y = 32945, map = 0 },   -- 說話之島 3F
    -- 銀騎士系
    [5]    = { x = 33084, y = 33391, map = 4 },   -- 銀騎士地下
    [6]    = { x = 33084, y = 33391, map = 4 },
    -- 古魯丁地下城
    [13]   = { x = 32613, y = 32775, map = 4 },
    [14]   = { x = 32613, y = 32775, map = 4 },
    -- 乘特地下城
    [15]   = { x = 33050, y = 32764, map = 4 },
    [16]   = { x = 33050, y = 32764, map = 4 },
    -- 火焰之影地下城
    [17]   = { x = 33050, y = 32764, map = 4 },
    -- 奇岩地下城
    [19]   = { x = 33429, y = 32814, map = 4 },
    [20]   = { x = 33429, y = 32814, map = 4 },
    -- 海音地下城
    [21]   = { x = 33600, y = 33240, map = 4 },
    [22]   = { x = 33600, y = 33240, map = 4 },
    [23]   = { x = 33600, y = 33240, map = 4 },
    -- 歐瑞地下城
    [24]   = { x = 34050, y = 32275, map = 4 },
    [25]   = { x = 34050, y = 32275, map = 4 },
    -- 忘卻之島
    [71]   = { x = 32575, y = 32945, map = 0 },
    [72]   = { x = 32575, y = 32945, map = 0 },
    -- 傲塔
    [101]  = { x = 34050, y = 32275, map = 4 },   -- 傲塔→歐瑞
    [102]  = { x = 34050, y = 32275, map = 4 },
    [103]  = { x = 34050, y = 32275, map = 4 },
    -- 象牙塔
    [26]   = { x = 34050, y = 32275, map = 4 },
    [27]   = { x = 34050, y = 32275, map = 4 },
    -- 亞丁地下城
    [28]   = { x = 34000, y = 33140, map = 4 },
    -- 新手區
    [2005] = { x = 32689, y = 32842, map = 2005 },
    -- 隱藏之谷
    [70]   = { x = 32579, y = 32735, map = 70 },
    -- 獸人森林
    [303]  = { x = 32744, y = 32447, map = 4 },
    -- 妖精森林
    [350]  = { x = 33050, y = 32340, map = 4 },
    -- 龍之谷
    [1005] = { x = 34050, y = 32275, map = 4 },
    [1011] = { x = 34050, y = 32275, map = 4 },
    [1017] = { x = 34050, y = 32275, map = 4 },
}

-- 計算兩點距離平方（省略開根號）
local function dist2(x1, y1, x2, y2)
    local dx = x1 - x2
    local dy = y1 - y2
    return dx * dx + dy * dy
end

-- 回家卷軸：依地圖和座標找最近城鎮（Java: GetbackTable + L1TownLocation）
function get_home_scroll_location(map_id, px, py)
    -- 非主大陸：查映射表
    if map_id ~= 4 then
        local loc = HOME_SCROLL_MAP[map_id]
        if loc then
            return loc
        end
        -- 未知地圖 fallback 到銀騎士
        return DEFAULT_RESPAWN
    end

    -- 主大陸（map 4）：找最近城鎮
    local best = nil
    local bestDist = math.huge
    for _, town in ipairs(TOWNS) do
        if town.map == 4 then
            local d = dist2(px, py, town.cx, town.cy)
            if d < bestDist then
                bestDist = d
                best = town
            end
        end
    end
    if best then
        return { x = best.rx, y = best.ry, map = 4 }
    end
    return DEFAULT_RESPAWN
end
