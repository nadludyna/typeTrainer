-- Learn's placement test: done or not, the personal speed goal (0 = the
-- config's) and the letters it found known.

ALTER TABLE progress ADD COLUMN placed BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE progress ADD COLUMN goal DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE progress ADD COLUMN known TEXT NOT NULL DEFAULT '';
