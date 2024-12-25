CREATE TABLE IF NOT EXISTS "store_products"(
    "id" UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    "product_id" UUID REFERENCES products(id) UNIQUE,
    "store_id" UUID REFERENCES stores(id),
    "product_material_code" INT REFERENCES products(material_code),
    "quantity" INT DEFAULT 0,
    "small_quantity" INT DEFAULT 0,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);