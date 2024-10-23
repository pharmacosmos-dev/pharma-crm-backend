CREATE TABLE IF NOT EXISTS "product_units" (
  "id" uuid PRIMARY KEY,
  "unit" varchar,
  "abbreviation" varchar,
  "accuracy" varchar,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
)