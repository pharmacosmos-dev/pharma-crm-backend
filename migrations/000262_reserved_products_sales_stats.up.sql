ALTER TABLE "reserved_products"
    ADD COLUMN IF NOT EXISTS "sold_quantity_15d"      BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "sold_quantity_prev_15d" BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "sold_change_percent"    NUMERIC(10, 2),
    ADD COLUMN IF NOT EXISTS "sold_calculated_at"     TIMESTAMP WITH TIME ZONE;
