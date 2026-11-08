-- Aborted rounds: dropped by Tab, kept raw for recalculation,
-- but never shown and never learned.

ALTER TABLE sessions ADD COLUMN aborted BOOLEAN NOT NULL DEFAULT false;
