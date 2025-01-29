CREATE SEQUENCE IF NOT EXISTS employees_public_id_seq START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS "employees" (
  "id" uuid PRIMARY KEY,
  "public_id" INTEGER NOT NULL DEFAULT nextval('employees_public_id_seq'),
  "store_id" uuid REFERENCES "stores"("id"),
  "role_id" uuid REFERENCES "roles"("id"),
  "first_name" varchar,
  "last_name" varchar,
  "full_name" VARCHAR,
  "phone" varchar,
  "email" varchar,
  "password" text,
  "language" varchar(10),
  "photo" VARCHAR(20),
  "created_by" uuid,
  "updated_by" uuid,
  "deleted_by" uuid,
  "is_active" boolean NOT NULL DEFAULT true,
  "gender" varchar(20),
  "status" varchar(20),
  "birthdate" DATE,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);