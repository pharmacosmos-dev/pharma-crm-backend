CREATE TABLE IF NOT EXISTS "category_products" (
    "id" UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "category_id" UUID REFERENCES "categories"("id") ON DELETE CASCADE,
    "product_id" UUID NOT NULL REFERENCES "products"(id) ON DELETE CASCADE,
    "is_open" BOOLEAN DEFAULT false,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
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
