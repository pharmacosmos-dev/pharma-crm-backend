CREATE TABLE IF NOT EXISTS "addresses" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "customer_id" UUID REFERENCES "customers"("id"),
  "country" VARCHAR(255),
  "city" VARCHAR(255),
  "address" VARCHAR(500),
  "postal_code" VARCHAR(100),
  "desc" VARCHAR(1000),
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "deleted_at" TIMESTAMP
);