-- Remove repricing tables (no longer needed)
DROP TABLE IF EXISTS online_price_revaluation_details;
DROP TABLE IF EXISTS online_price_revaluations;

-- Redesign online_store_products as append-only history table
ALTER TABLE online_store_products
    DROP CONSTRAINT IF EXISTS online_store_products_store_id_product_id_type_key,
    DROP COLUMN IF EXISTS old_supply_price,
    DROP COLUMN IF EXISTS updated_at,
    ADD COLUMN IF NOT EXISTS material_code VARCHAR(100);

-- Index for fast latest-price lookup per product
CREATE INDEX IF NOT EXISTS idx_osp_product_store_type_created
    ON online_store_products (product_id, store_id, type, created_at DESC);
