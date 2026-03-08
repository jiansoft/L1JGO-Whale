-- +goose Up

-- 城堡動態狀態（稅率、寶庫、攻城時間）
CREATE TABLE castles (
    castle_id    INT PRIMARY KEY,
    castle_name  VARCHAR(32) NOT NULL,
    tax_rate     INT NOT NULL DEFAULT 10,
    public_money BIGINT NOT NULL DEFAULT 0,
    war_time     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO castles (castle_id, castle_name, tax_rate, public_money, war_time) VALUES
    (1, '肯特',   10, 0, NOW() + INTERVAL '7 days'),
    (2, '妖魔',   10, 0, NOW() + INTERVAL '7 days'),
    (3, '風木',   10, 0, NOW() + INTERVAL '7 days'),
    (4, '奇巖',   10, 0, NOW() + INTERVAL '7 days'),
    (5, '海音',   10, 0, NOW() + INTERVAL '7 days'),
    (6, '侏儒',   10, 0, NOW() + INTERVAL '7 days'),
    (7, '亞丁',   10, 0, NOW() + INTERVAL '7 days'),
    (8, '狄亞得', 10, 0, NOW() + INTERVAL '7 days');

-- +goose Down
DROP TABLE IF EXISTS castles;
