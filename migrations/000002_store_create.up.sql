CREATE TABLE IF NOT EXISTS "stores" (
  "id" uuid PRIMARY KEY,
  "name" varchar(255),
  "location" varchar(255),
  "created_by" uuid,
  "updated_by" uuid,
  "deleted_by" uuid,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);