CREATE SEQUENCE IF NOT EXISTS employees_public_id_seq START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS "employees" (
  "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  "public_id" INTEGER NOT NULL DEFAULT nextval('employees_public_id_seq'),
  "store_id" uuid REFERENCES "stores"("id"),
  "first_name" VARCHAR(255),
  "last_name" VARCHAR(255),
  "full_name" VARCHAR(255),
  "phone" VARCHAR(20),
  "email" VARCHAR(255),
  "password" VARCHAR(500),
  "language" varchar(10),
  "photo" VARCHAR(20),
  "is_active" BOOLEAN NOT NULL DEFAULT TRUE,
  "gender" VARCHAR(20),
  "status" VARCHAR(20),
  "birthdate" DATE,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "deleted_at" TIMESTAMP
);