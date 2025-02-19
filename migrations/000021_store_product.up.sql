CREATE TABLE IF NOT EXISTS "store_products"(
    "id" UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "product_id" UUID REFERENCES products(id) ON DELETE CASCADE,
    "store_id" UUID REFERENCES stores(id) ON DELETE CASCADE,
    "pack_quantity" INT DEFAULT 0,
    "unit_quantity" INT DEFAULT 0,
    "small_quantity" INT DEFAULT 0,
    "retail_price" NUMERIC(10, 2) DEFAULT 0,
    "supply_price" NUMERIC(10, 2) DEFAULT 0,
    "bonus_percent" INT DEFAULT 0,
    "vat" INT DEFAULT 0,
    "markup" INT DEFAULT 0,
    "expire_date" DATE,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);