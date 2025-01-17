CREATE TABLE IF NOT EXISTS "addresses" (
  "id" uuid PRIMARY KEY,
  "customer_id" uuid,
  "country" varchar,
  "city" varchar,
  "address" varchar,
  "postal_code" varchar,
  "desc" varchar,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);