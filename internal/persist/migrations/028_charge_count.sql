-- +goose Up

ALTER TABLE character_items ADD COLUMN charge_count SMALLINT NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE character_items DROP COLUMN IF EXISTS charge_count;
