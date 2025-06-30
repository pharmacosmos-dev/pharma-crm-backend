CREATE TABLE IF NOT EXISTS "categories" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "category_id" UUID REFERENCES "categories"("id") ON DELETE CASCADE,
  "name" VARCHAR(255),
  "photo" VARCHAR(255),
  "is_active" BOOLEAN NOT NULL DEFAULT true,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "deleted_at" TIMESTAMP
);