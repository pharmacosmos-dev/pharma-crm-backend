CREATE TABLE IF NOT EXISTS "online_store_products" (
    "id"              UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "store_id"        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    "product_id"      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    "type"            VARCHAR(50) NOT NULL,
    "retail_price"    NUMERIC(10, 2) NOT NULL DEFAULT 0,
    "supply_price"    NUMERIC(10, 2) NOT NULL DEFAULT 0,
    "old_supply_price" NUMERIC(10, 2) NOT NULL DEFAULT 0,
    "created_by"      UUID REFERENCES employees(id) ON DELETE SET NULL,
    "created_at"      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at"      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE ("store_id", "product_id", "type")
);
