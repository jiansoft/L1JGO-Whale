-- item/durability.lua — Weapon durability damage formulas
-- Java reference: L1Attack.damageNpcWeaponDurability(), L1Inventory.receiveDamage

-- calc_durability_damage(ctx) -> {should_damage, max_durability}
-- ctx = {enchant_lvl, bless, current_durability}
-- bless: 0=normal, 1=blessed, 2=cursed
--
-- Java formula:
--   Max durability = enchant + 5 (minimum 5)
--   Damage probability: blessed = 3%, normal/cursed = 10%
function calc_durability_damage(ctx)
    -- Max durability = enchant + 5
    local max_dur = ctx.enchant_lvl + 5
    if max_dur < 5 then
        max_dur = 5
    end

    -- Already at max damage
    if ctx.current_durability >= max_dur then
        return { should_damage = false, max_durability = max_dur }
    end

    -- Probability: blessed = 3%, else = 10%
    local threshold = 10
    if ctx.bless == 1 then
        threshold = 3
    end

    local roll = math.random(1, 100)
    local should_damage = roll <= threshold

    return { should_damage = should_damage, max_durability = max_dur }
end
