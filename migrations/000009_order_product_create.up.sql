CREATE TABLE IF NOT EXISTS "order_products" (
  "id" uuid PRIMARY KEY,
  "product_id" uuid,
  "order_id" uuid,
  "created_by" uuid,
  "count" int,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);