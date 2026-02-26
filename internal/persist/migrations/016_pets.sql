-- +goose Up

-- Pet system tables

-- character_pets: persistent pet data, keyed by amulet/collar item object ID.
-- One row per tamed pet. The amulet item in the player's inventory IS the pet.
CREATE TABLE IF NOT EXISTS character_pets (
    item_obj_id  INT PRIMARY KEY,            -- FK to items.object_id (amulet/collar)
    obj_id       INT NOT NULL DEFAULT 0,     -- Pet instance object ID
    npc_id       INT NOT NULL DEFAULT 0,     -- Base NPC template ID (determines species)
    name         VARCHAR(45) NOT NULL DEFAULT '',
    level        SMALLINT NOT NULL DEFAULT 1,
    hp           INT NOT NULL DEFAULT 0,     -- Current/Max HP (snapshot at save)
    mp           INT NOT NULL DEFAULT 0,     -- Current/Max MP (snapshot at save)
    exp          INT NOT NULL DEFAULT 0,     -- Experience points
    lawful       INT NOT NULL DEFAULT 0      -- Alignment
);

CREATE INDEX IF NOT EXISTS idx_character_pets_npc ON character_pets (npc_id);
