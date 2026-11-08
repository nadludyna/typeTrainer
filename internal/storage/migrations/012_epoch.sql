-- Epoch: bumped on every reset or import, stamped on every new session,
-- so records saved before a wipe can be told apart from records after it.

ALTER TABLE sessions ADD COLUMN epoch INT NOT NULL DEFAULT 0;
CREATE TABLE epochs (
    user_id TEXT   NOT NULL,
    lang    TEXT   NOT NULL,
    epoch   INT    NOT NULL,
    PRIMARY KEY (user_id, lang)
);
