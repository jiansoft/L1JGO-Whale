-- +goose Up
CREATE TABLE character_buffs (
    char_id         INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    skill_id        INTEGER NOT NULL,
    remaining_time  INTEGER NOT NULL DEFAULT 0,
    poly_id         INTEGER NOT NULL DEFAULT 0,
    delta_ac        SMALLINT NOT NULL DEFAULT 0,
    delta_str       SMALLINT NOT NULL DEFAULT 0,
    delta_dex       SMALLINT NOT NULL DEFAULT 0,
    delta_con       SMALLINT NOT NULL DEFAULT 0,
    delta_wis       SMALLINT NOT NULL DEFAULT 0,
    delta_intel     SMALLINT NOT NULL DEFAULT 0,
    delta_cha       SMALLINT NOT NULL DEFAULT 0,
    delta_max_hp    SMALLINT NOT NULL DEFAULT 0,
    delta_max_mp    SMALLINT NOT NULL DEFAULT 0,
    delta_hit_mod   SMALLINT NOT NULL DEFAULT 0,
    delta_dmg_mod   SMALLINT NOT NULL DEFAULT 0,
    delta_sp        SMALLINT NOT NULL DEFAULT 0,
    delta_mr        SMALLINT NOT NULL DEFAULT 0,
    delta_hpr       SMALLINT NOT NULL DEFAULT 0,
    delta_mpr       SMALLINT NOT NULL DEFAULT 0,
    delta_bow_hit   SMALLINT NOT NULL DEFAULT 0,
    delta_bow_dmg   SMALLINT NOT NULL DEFAULT 0,
    delta_fire_res  SMALLINT NOT NULL DEFAULT 0,
    delta_water_res SMALLINT NOT NULL DEFAULT 0,
    delta_wind_res  SMALLINT NOT NULL DEFAULT 0,
    delta_earth_res SMALLINT NOT NULL DEFAULT 0,
    delta_dodge     SMALLINT NOT NULL DEFAULT 0,
    set_move_speed  SMALLINT NOT NULL DEFAULT 0,
    set_brave_speed SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (char_id, skill_id)
);

-- +goose Down
DROP TABLE IF EXISTS character_buffs;
