CREATE TABLE IF NOT EXISTS "tags" (
  "id" uuid PRIMARY KEY,
  "name" varchar,
  "type" varchar,
  "status" varchar,
  "desc" text,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);