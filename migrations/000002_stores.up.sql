CREATE TABLE IF NOT EXISTS "stores" (
  "id" uuid PRIMARY KEY,
  "store_code" int UNIQUE,
  "name" varchar(255),
  "location" varchar(255),
  "address" varchar(255),
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);