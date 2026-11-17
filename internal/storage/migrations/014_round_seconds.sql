-- The time a round took, as the browser measured it (pauses left out):
-- the daily goal counts it, Tab-dropped rounds included. Older rows have
-- none and fall back to an estimate from speed and characters.

ALTER TABLE sessions ADD COLUMN seconds DOUBLE PRECISION;
