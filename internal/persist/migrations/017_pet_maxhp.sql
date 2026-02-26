-- +goose Up

-- Add max HP/MP columns to character_pets for proper persistence.
-- Previously HP/MP stored combined current/max (lost when pet was dead).
ALTER TABLE character_pets ADD COLUMN IF NOT EXISTS hpmax INT NOT NULL DEFAULT 0;
ALTER TABLE character_pets ADD COLUMN IF NOT EXISTS mpmax INT NOT NULL DEFAULT 0;
