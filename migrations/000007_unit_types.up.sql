CREATE TABLE IF NOT EXISTS "unit_types"(
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "unit_name" VARCHAR(50) NOT NULL,
  "codename" VARCHAR(50),
  "short_name" VARCHAR(10),
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "deleted_at" TIMESTAMP
);