CREATE TABLE IF NOT EXISTS "category_products" (
    "id" UUID NOT NULL PRIMARY KEY,
    "category_id" UUID NOT NULL REFERENCES "categories"(id),
    "product_id" UUID NOT NULL REFERENCES "products"(id),
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);