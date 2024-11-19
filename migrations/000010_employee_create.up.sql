CREATE TABLE IF NOT EXISTS "employees" (
  "id" uuid PRIMARY KEY,
  "client_type_id" uuid,
  "role_id" uuid,
  "first_name" varchar,
  "last_name" varchar,
  "phone" varchar,
  "email" varchar,
  "password" text,
  "language" varchar(10),
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);