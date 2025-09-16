ALTER TABLE IF EXISTS products
    DROP COLUMN IF EXISTS last_updated_at;

ALTER TABLE IF EXISTS store_products
    DROP COLUMN IF EXISTS last_updated_at;
