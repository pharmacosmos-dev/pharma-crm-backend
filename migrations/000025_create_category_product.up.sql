CREATE TABLE IF NOT EXISTS "category_products" (
    "id" UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    "category_id" UUID NOT NULL REFERENCES "categories"(id),
    "product_id" UUID NOT NULL REFERENCES "products"(id),
    "is_open" BOOLEAN DEFAULT false,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE "category_products"
ADD CONSTRAINT 
    category_products_product_id_category_id_key 
UNIQUE ("product_id", "category_id");
