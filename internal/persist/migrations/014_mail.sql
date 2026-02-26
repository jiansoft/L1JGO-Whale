-- +goose Up
CREATE TABLE mail (
    id          SERIAL PRIMARY KEY,
    type        SMALLINT NOT NULL DEFAULT 0,
    sender      VARCHAR(45) NOT NULL,
    receiver    VARCHAR(45) NOT NULL,
    date        TIMESTAMP NOT NULL DEFAULT NOW(),
    read_status SMALLINT NOT NULL DEFAULT 0,
    inbox_id    INTEGER NOT NULL,
    subject     BYTEA,
    content     BYTEA
);
CREATE INDEX idx_mail_inbox ON mail(inbox_id, type);

-- +goose Down
DROP TABLE IF EXISTS mail;
