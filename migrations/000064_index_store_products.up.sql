CREATE INDEX IF NOT EXISTS idx_store_products_covering
ON store_products(store_id, pack_quantity, unit_quantity)
INCLUDE (product_id);