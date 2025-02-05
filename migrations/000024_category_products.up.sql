CREATE TABLE IF NOT EXISTS "category_products" (
    "id" UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    "category_id" uuid REFERENCES "categories"("id") ON DELETE CASCADE,
    "product_id" UUID NOT NULL REFERENCES "products"(id),
    "is_open" BOOLEAN DEFAULT false,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    -- Check if the constraint exists
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_constraint 
        WHERE conname = 'category_products_product_id_category_id_key'
    ) THEN
        -- Add the unique constraint
        ALTER TABLE "category_products"
        ADD CONSTRAINT category_products_product_id_category_id_key 
        UNIQUE ("product_id", "category_id");
    END IF;
END $$;
