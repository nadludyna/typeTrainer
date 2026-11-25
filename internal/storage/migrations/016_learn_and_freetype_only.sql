-- The site keeps Learn and Freetype only: Climb's rounds, the study group
-- and the old typing test's flag go.
DELETE FROM sessions WHERE mode = 'freeplay';
ALTER TABLE sessions DROP CONSTRAINT sessions_mode_check;
ALTER TABLE sessions ADD CONSTRAINT sessions_mode_check
    CHECK (mode IN ('learning', 'freetype-sentences', 'freetype-custom')
           OR mode ~ '^freetype-[0-9]+(-long)?$');
ALTER TABLE sessions DROP COLUMN study_group;
ALTER TABLE progress DROP COLUMN study_group;
ALTER TABLE progress DROP COLUMN test_passed;
