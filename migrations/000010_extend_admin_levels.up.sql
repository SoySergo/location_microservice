-- Расширяем диапазон поддерживаемых admin_level с (2,4,6,8,9) до (2-11)

ALTER TABLE admin_boundaries 
    DROP CONSTRAINT IF EXISTS admin_boundaries_admin_level_check;

ALTER TABLE admin_boundaries 
    ADD CONSTRAINT admin_boundaries_admin_level_check 
    CHECK (admin_level >= 2 AND admin_level <= 11);
