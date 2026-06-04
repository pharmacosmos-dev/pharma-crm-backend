ALTER TABLE transfers ADD COLUMN IF NOT EXISTS "driver_name" VARCHAR(255) DEFAULT NULL;

UPDATE transfers SET driver_name = driver_store_b WHERE driver_store_b IS NOT NULL AND driver_store_b != '';
