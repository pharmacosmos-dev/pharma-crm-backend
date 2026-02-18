CREATE TABLE IF NOT EXISTS product_price_changed (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    store_code INT NOT NULL,
    product_id UUID NOT NULL,
    max_price NUMERIC(18, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_product_price_changed_store_product
ON product_price_changed(store_id, product_id);

CREATE INDEX IF NOT EXISTS idx_product_price_changed_store_id
ON product_price_changed(store_id);
