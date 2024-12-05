CREATE TABLE IF NOT EXISTS "employees" (
  "id" uuid PRIMARY KEY,
  "store_id" uuid REFERENCES "stores"("id"),
  "role_id" uuid REFERENCES "roles"("id"),
  "first_name" varchar,
  "last_name" varchar,
  "phone" varchar,
  "email" varchar,
  "password" text,
  "language" varchar(10),
  "photo" VARCHAR(20),
  "created_by" uuid,
  "updated_by" uuid,
  "deleted_by" uuid,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);