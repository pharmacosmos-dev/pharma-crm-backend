UPDATE transfers SET driver_store_b = driver_name WHERE driver_name IS NOT NULL AND driver_name != '' AND (driver_store_b IS NULL OR driver_store_b = '');

ALTER TABLE transfers DROP COLUMN IF EXISTS "driver_name";
