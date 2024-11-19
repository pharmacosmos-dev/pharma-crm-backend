CREATE TABLE IF NOT EXISTS "brands" (
  "id" uuid PRIMARY KEY,
  "name" varchar(255),
  "logo" text,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);