DROP INDEX IF EXISTS marking_unit_sales_store_product_id_idx;
DROP INDEX IF EXISTS marking_unit_sales_store_id_idx;

ALTER TABLE marking_unit_sales
    DROP COLUMN IF EXISTS "store_product_id",
    DROP COLUMN IF EXISTS "store_id";
