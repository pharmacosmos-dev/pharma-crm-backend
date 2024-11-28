CREATE TABLE IF NOT EXISTS "categories" (
  "id" uuid PRIMARY KEY,
  "category_id" uuid REFERENCES "categories"("id"),
  "name" varchar(255),
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);