ALTER TABLE "reserved_products"
    DROP COLUMN IF EXISTS "sold_quantity_15d",
    DROP COLUMN IF EXISTS "sold_quantity_prev_15d",
    DROP COLUMN IF EXISTS "sold_change_percent",
    DROP COLUMN IF EXISTS "sold_calculated_at";
