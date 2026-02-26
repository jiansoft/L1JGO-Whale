-- +goose Up
-- Weapon durability: 0 = perfect, higher = more damaged (range 0-127).
-- Repair NPC sets to 0; combat damage increments by 1.
ALTER TABLE character_items ADD COLUMN durability SMALLINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE character_items DROP COLUMN durability;
