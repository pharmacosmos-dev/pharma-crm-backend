ALTER TABLE marking_unit_sales
    ADD COLUMN IF NOT EXISTS "store_id" UUID REFERENCES "stores" ("id"),
    ADD COLUMN IF NOT EXISTS "store_product_id" UUID REFERENCES "store_products" ("id") ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS marking_unit_sales_store_id_idx ON marking_unit_sales (store_id);
CREATE INDEX IF NOT EXISTS marking_unit_sales_store_product_id_idx ON marking_unit_sales (store_product_id);
