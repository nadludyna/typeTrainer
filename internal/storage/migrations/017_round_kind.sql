-- The kind of a round: '' the usual one, 'weak' a round built from the
-- typist's weak spots, 'clean' a round that ends at the third mistake.
-- A round of a kind is kept and shown but moves no level.

ALTER TABLE sessions ADD COLUMN kind TEXT NOT NULL DEFAULT '';
