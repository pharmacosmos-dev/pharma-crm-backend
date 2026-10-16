
CREATE TABLE IF NOT EXISTS marking_unit_sales(
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "marking" VARCHAR(500) NOT NULL UNIQUE,
    "product_id" UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    "unit_per_pack" INT NOT NULL DEFAULT 1 CHECK ("unit_per_pack" > 0),
    "sold_quantity" INT NOT NULL DEFAULT 0 CHECK ("sold_quantity" >= 0 AND "sold_quantity" <= "unit_per_pack"),
    "stock_quantity" INT NOT NULL DEFAULT 0 CHECK ("stock_quantity" >= 0),
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS marking_unit_sales_product_id_idx ON marking_unit_sales (product_id);
