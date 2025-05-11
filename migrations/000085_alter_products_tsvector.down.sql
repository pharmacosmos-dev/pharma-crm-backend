ALTER TABLE products DROP COLUMN IF EXISTS name_tsvector;
DROP INDEX IF EXISTS idx_products_name_tsvector;