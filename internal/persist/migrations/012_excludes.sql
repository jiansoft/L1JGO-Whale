-- +goose Up
CREATE TABLE character_excludes (
    id         SERIAL PRIMARY KEY,
    char_id    INTEGER NOT NULL,
    exclude_name VARCHAR(45) NOT NULL,
    UNIQUE(char_id, exclude_name)
);
CREATE INDEX idx_excludes_char ON character_excludes(char_id);

-- +goose Down
DROP TABLE IF EXISTS character_excludes;
