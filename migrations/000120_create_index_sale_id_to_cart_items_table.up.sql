CREATE INDEX IF NOT EXISTS idx_cart_items_sale_id
    ON cart_items(sale_id);