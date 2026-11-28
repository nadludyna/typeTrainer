-- Learn's accuracy rules: a letter's last presses (1 = correct, oldest
-- first) and whether it slipped under the lower threshold
--.

ALTER TABLE key_stats ADD COLUMN hits TEXT NOT NULL DEFAULT '';
ALTER TABLE key_stats ADD COLUMN slipped BOOLEAN NOT NULL DEFAULT FALSE;
