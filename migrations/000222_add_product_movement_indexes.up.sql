CREATE INDEX IF NOT EXISTS idx_import_details_import_product
ON import_details (import_id, product_id);

CREATE INDEX IF NOT EXISTS idx_cart_items_store_product_id
ON cart_items (store_product_id);
