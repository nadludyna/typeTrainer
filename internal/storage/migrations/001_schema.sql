-- The whole database. A change to it is a new numbered file next to this
-- one, applied in order at start (migrate in postgres.go), never an edit here.

CREATE TABLE progress (
    user_id  TEXT    PRIMARY KEY,
    letters  JSONB   NOT NULL DEFAULT '{}', -- letter → engine.KeyStat
    unlocked INTEGER NOT NULL DEFAULT 0,
    lap      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE rounds (
    id          BIGSERIAL        PRIMARY KEY,
    user_id     TEXT             NOT NULL,
    round_id    TEXT,                       -- the browser's id: a round sent twice is kept once
    mode        TEXT             NOT NULL,  -- learn, reinforce, freetype-<list>
    kind        TEXT             NOT NULL DEFAULT '',
    level       INTEGER          NOT NULL DEFAULT 0,
    lap         INTEGER          NOT NULL DEFAULT 0,
    length      TEXT             NOT NULL,
    wpm         DOUBLE PRECISION NOT NULL,
    accuracy    DOUBLE PRECISION NOT NULL,
    chars       INTEGER          NOT NULL,
    consistency DOUBLE PRECISION,
    unlocked    INTEGER          NOT NULL DEFAULT 0,
    finished_at TIMESTAMPTZ      NOT NULL,
    keys        JSONB
);
CREATE UNIQUE INDEX rounds_once ON rounds (user_id, round_id);
CREATE INDEX rounds_by_user ON rounds (user_id, id);
