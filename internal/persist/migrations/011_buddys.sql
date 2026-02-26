-- +goose Up
CREATE TABLE character_buddys (
    id         SERIAL PRIMARY KEY,
    char_id    INTEGER NOT NULL,
    buddy_id   INTEGER NOT NULL,
    buddy_name VARCHAR(45) NOT NULL,
    UNIQUE(char_id, buddy_id)
);
CREATE INDEX idx_buddys_char ON character_buddys(char_id);

-- +goose Down
DROP TABLE IF EXISTS character_buddys;
