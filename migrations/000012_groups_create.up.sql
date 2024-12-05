CREATE TABLE IF NOT EXISTS "groups" (
  "id" uuid PRIMARY KEY,
  "name" varchar,
  "discount_percent" int,
  "is_discount" bool,
  "is_public" bool,
  "desc" text,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);