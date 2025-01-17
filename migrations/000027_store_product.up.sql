CREATE TABLE IF NOT EXISTS "store_products"(
    "id" UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    "product_id" UUID REFERENCES products(id) UNIQUE,
    "store_id" UUID REFERENCES stores(id),
    "product_material_code" INT REFERENCES products(material_code),
    "quantity" INT DEFAULT 0,
    "pack_quantity" INT DEFAULT 0,
    "unit_quantity" INT DEFAULT 0,
    "unit_per_pack" INTEGER DEFAULT 0,
    "small_quantity" INT DEFAULT 0,
    "retail_price" NUMERIC(10, 2) DEFAULT 0,
    "supply_price" NUMERIC(10, 2) DEFAULT 0,
    "vat" INT DEFAULT 0,
    "expire_date" DATE,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);