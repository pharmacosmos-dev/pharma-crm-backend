DROP INDEX IF EXISTS idx_osp_product_store_type_created;

ALTER TABLE online_store_products
    DROP COLUMN IF EXISTS material_code,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS old_supply_price NUMERIC(10, 2) NOT NULL DEFAULT 0;

ALTER TABLE online_store_products
    ADD CONSTRAINT online_store_products_store_id_product_id_type_key UNIQUE (store_id, product_id, type);
