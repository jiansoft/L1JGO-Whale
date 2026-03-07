-- 物品元素屬性強化欄位
-- kind: 0=無, 1=地, 2=火, 4=水, 8=風
-- level: 0-5（強化階段）
ALTER TABLE character_items ADD COLUMN IF NOT EXISTS attr_enchant_kind SMALLINT NOT NULL DEFAULT 0;
ALTER TABLE character_items ADD COLUMN IF NOT EXISTS attr_enchant_level SMALLINT NOT NULL DEFAULT 0;
