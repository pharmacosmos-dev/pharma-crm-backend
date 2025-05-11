DROP INDEX IF EXISTS idx_store_products_covering;

CREATE INDEX IF NOT EXISTS idx_store_products_product_id ON store_products(product_id);