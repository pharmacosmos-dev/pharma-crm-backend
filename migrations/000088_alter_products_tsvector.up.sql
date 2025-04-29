ALTER TABLE products ADD COLUMN IF NOT EXISTS name_tsvector tsvector;
CREATE INDEX IF NOT EXISTS idx_products_name_tsvector ON products USING gin(name_tsvector);