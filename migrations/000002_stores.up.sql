CREATE TABLE IF NOT EXISTS "stores" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "store_code" INT UNIQUE,
  "name" VARCHAR(255),
  "location" VARCHAR(255),
  "address" VARCHAR(255),
  "is_active" BOOLEAN NOT NULL DEFAULT true,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "deleted_at" TIMESTAMP
);