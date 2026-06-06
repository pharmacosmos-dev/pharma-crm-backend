ALTER TABLE online_products_price
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS updated_by;
