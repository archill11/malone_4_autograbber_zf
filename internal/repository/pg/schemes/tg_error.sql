CREATE TABLE IF NOT EXISTS tg_errors (
    bot_id            BIGINT,
    bot_token         TEXT,
    bot_username      TEXT,
    bot_ch_id         BIGINT DEFAULT 0,
    err_description   TEXT DEFAULT '',
    err_count         INT DEFAULT 0,

    PRIMARY KEY (bot_id, err_description)
);

-------------------------------------------
