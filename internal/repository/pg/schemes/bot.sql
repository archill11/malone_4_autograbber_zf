CREATE TABLE IF NOT EXISTS bots (
    id                BIGINT,
    token             TEXT,
    username          TEXT,
    first_name        TEXT,
    is_donor          INT,
    ch_id             BIGINT    DEFAULT 0,
    ch_link           TEXT      DEFAULT '',
    group_link_id     INT       DEFAULT 0,
    lichka            TEXT      DEFAULT '',
    linked_lichka     TEXT      DEFAULT '',
    user_creator      BIGINT    DEFAULT 0,
    is_disable        INT       DEFAULT 0,
    created_at        TIMESTAMP DEFAULT now(),
    ch_is_skam        INT       DEFAULT 0,
    personal_link     TEXT      DEFAULT '',
    donor_ch_id       BIGINT    DEFAULT 0,
    is_err_in_stat    INT       DEFAULT 0,

    to_click_short_link           TEXT DEFAULT '',
    to_click_short_link_to_lichka TEXT DEFAULT '',
    short_domen_to_replace        TEXT DEFAULT '',

    additional_chs JSONB DEFAULT '[]',

    PRIMARY KEY (id, token)
);

-------------------------------------------

ALTER TABLE bots
  ADD COLUMN IF NOT EXISTS short_domen_to_replace TEXT DEFAULT '';
-------------------------------------------

ALTER TABLE bots
  ADD COLUMN IF NOT EXISTS additional_chs JSONB DEFAULT '[]';

ALTER TABLE bots
  ADD COLUMN IF NOT EXISTS linked_lichka TEXT DEFAULT '';
