-- +goose Up
CREATE TABLE server_board (
    id       SERIAL PRIMARY KEY,
    name     VARCHAR(45) NOT NULL,
    date     VARCHAR(20) NOT NULL,
    title    VARCHAR(255) NOT NULL,
    content  TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS server_board;
