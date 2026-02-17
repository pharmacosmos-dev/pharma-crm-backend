CREATE TABLE IF NOT EXISTS "store_product_stocks" (
    "id"            BIGSERIAL    PRIMARY KEY,
    "store_id"      UUID         REFERENCES "stores"("id") ON DELETE CASCADE,
    "product_id"    UUID         REFERENCES "products"("id") ON DELETE CASCADE,
    "unit_quantity" BIGINT       DEFAULT 0,
    "min_price"     DECIMAL(10, 2) DEFAULT 0.00,
    "max_price"     DECIMAL(10, 2) DEFAULT 0.00,
    "created_at"    TIMESTAMP    NOT NULL DEFAULT NOW(),
    "updated_at"    TIMESTAMP    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_store_product_stocks_store_id ON "store_product_stocks"("store_id");
CREATE INDEX IF NOT EXISTS idx_store_product_stocks_product_id ON "store_product_stocks"("product_id");
CREATE UNIQUE INDEX IF NOT EXISTS idx_store_product_stocks_store_product_id ON "store_product_stocks"("store_id", "product_id");