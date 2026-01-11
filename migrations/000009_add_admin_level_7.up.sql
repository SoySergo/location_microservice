-- Add admin_level 7 support for comarcas (counties)
ALTER TABLE admin_boundaries DROP CONSTRAINT admin_boundaries_admin_level_check;
ALTER TABLE admin_boundaries ADD CONSTRAINT admin_boundaries_admin_level_check 
    CHECK (admin_level IN (2, 4, 6, 7, 8, 9, 10));
