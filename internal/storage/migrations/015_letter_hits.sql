-- Learn's accuracy rules: a letter's last presses (1 = correct, oldest
-- first) and whether it slipped under the lower threshold
-- (docs/HOW-THE-MODES-WORK.md, section 14).

ALTER TABLE key_stats ADD COLUMN hits TEXT NOT NULL DEFAULT '';
ALTER TABLE key_stats ADD COLUMN slipped BOOLEAN NOT NULL DEFAULT FALSE;
