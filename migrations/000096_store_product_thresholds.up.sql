CREATE TABLE IF NOT EXISTS store_product_thresholds (
    "id" SERIAL PRIMARY KEY,
    "store_id" UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    "product_id" UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    "kvant" INTEGER DEFAULT 1,
    "min_quantity" NUMERIC(10,2) NOT NULL DEFAULT 0,
    "max_quantity" NUMERIC(10,2) NOT NULL DEFAULT 0,
    "created_at" TIMESTAMP DEFAULT NOW(),
    "updated_at" TIMESTAMP DEFAULT NOW(),
    UNIQUE (store_id, product_id)
);
