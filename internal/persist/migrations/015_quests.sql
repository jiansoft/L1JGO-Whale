-- +goose Up

-- Character quest progress table.
-- CLAUDE.md: "character_quests = 獨立資料表（非 JSONB——需要查詢與 migration）"
-- Each row tracks one quest per character. Designed for efficient querying
-- of quest state without JSONB parsing.
CREATE TABLE character_quests (
    id          BIGSERIAL PRIMARY KEY,
    char_id     INT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    quest_id    INT NOT NULL,                    -- quest template ID
    step        INT NOT NULL DEFAULT 0,          -- current quest step (0 = just accepted)
    status      SMALLINT NOT NULL DEFAULT 0,     -- 0=in_progress, 1=completed, 2=failed
    kill_count  INT NOT NULL DEFAULT 0,          -- tracked mob kills
    item_count  INT NOT NULL DEFAULT 0,          -- tracked item collection
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    UNIQUE(char_id, quest_id)
);

CREATE INDEX idx_character_quests_char ON character_quests(char_id);
CREATE INDEX idx_character_quests_active ON character_quests(char_id, status) WHERE status = 0;

-- +goose Down

DROP TABLE IF EXISTS character_quests;
