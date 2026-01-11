-- Add admin_level 10 support for micro-districts (barrios)
ALTER TABLE admin_boundaries DROP CONSTRAINT admin_boundaries_admin_level_check;
ALTER TABLE admin_boundaries ADD CONSTRAINT admin_boundaries_admin_level_check 
    CHECK (admin_level IN (2, 4, 6, 8, 9, 10));
