-- Откат расширения admin_level - возвращаем к (2,4,6,8,9)

ALTER TABLE admin_boundaries 
    DROP CONSTRAINT IF EXISTS admin_boundaries_admin_level_check;

ALTER TABLE admin_boundaries 
    ADD CONSTRAINT admin_boundaries_admin_level_check 
    CHECK (admin_level IN (2, 4, 6, 8, 9));
