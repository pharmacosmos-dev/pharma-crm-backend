CREATE TYPE  "product_type_enum" AS ENUM (
  'product',
  'service',
  'set'
);

CREATE TYPE  "product_variability" AS ENUM (
  'basic',
  'variative'
);

CREATE TYPE  "product_status" AS ENUM (
  'active',
  'inactive',
  'low_stock',
  'zero_stock'
);

CREATE TYPE  "order_status" AS ENUM (
  'pending',
  'completed',
  'canceled'
);

CREATE TABLE IF NOT EXISTS "stores" (
  "id" uuid PRIMARY KEY,
  "name" varchar(255),
  "location" varchar(255),
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "categories" (
  "id" uuid PRIMARY KEY,
  "name" varchar(255),
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "brands" (
  "id" uuid PRIMARY KEY,
  "name" varchar(255),
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "suppliers" (
  "id" uuid PRIMARY KEY,
  "name" varchar(255),
  "phone" varchar[],
  "desc" text,
  "company_legal_name" varchar(255),
  "legal_address" varchar(255),
  "country" varchar(255),
  "zip_code" varchar,
  "bank_account" varchar,
  "bank_name" varchar,
  "bank_tin" varchar,
  "bank_ibt" varchar,
  "file_url" varchar,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "products" (
  "id" uuid PRIMARY KEY,
  "store_id" uuid,
  "category_id" uuid,
  "brand_id" uuid,
  "supplier_id" uuid,
  "product_type" product_type_enum,
  "product_variability" product_variability,
  "name" varchar(255),
  "sku" varchar(255) UNIQUE,
  "barcode" varchar(255),
  "unit" varchar(255),
  "main_photo" varchar,
  "photos" varchar[],
  "supply_price" int,
  "markup" int,
  "retail_price" int,
  "quantity" int,
  "desc" text,
  "status" product_status,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "orders" (
  "id" uuid PRIMARY KEY,
  "supplier_id" uuid,
  "store_id" uuid,
  "created_by" uuid,
  "name" varchar,
  "order_amount" int,
  "order_retail_amount" int,
  "payment_amount" int,
  "payment_dept_amount" int,
  "shipment_date" date,
  "status" order_status,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "order_products" (
  "id" uuid PRIMARY KEY,
  "product_id" uuid,
  "order_id" uuid,
  "created_by" uuid,
  "count" int,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "employees" (
  "id" uuid PRIMARY KEY,
  "client_type_id" uuid,
  "role_id" uuid,
  "first_name" varchar,
  "last_name" varchar,
  "phone" varchar[],
  "email" varchar,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "client_types" (
  "id" uuid PRIMARY KEY,
  "name" varchar,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "roles" (
  "id" uuid PRIMARY KEY,
  "name" varchar,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "groups" (
  "id" uuid PRIMARY KEY,
  "name" varchar,
  "discount_percent" int,
  "is_discount" bool,
  "is_public" bool,
  "desc" text,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "tags" (
  "id" uuid PRIMARY KEY,
  "name" varchar,
  "type" varchar,
  "status" varchar,
  "desc" text,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "customers" (
  "id" uuid PRIMARY KEY,
  "group_id" uuid,
  "tag_id" uuid,
  "first_name" varchar,
  "last_name" varchar,
  "middle_name" varchar,
  "phone" varchar[],
  "birthday" date,
  "gender" varchar,
  "marital_status" varchar,
  "primary_lang" varchar,
  "email" varchar,
  "tg_username" varchar,
  "facebook" varchar,
  "instagram" varchar,
  "is_sms_notify" bool,
  "is_phone_notify" bool,
  "is_social_notify" bool,
  "is_email_notify" bool,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE IF NOT EXISTS "addresses" (
  "id" uuid PRIMARY KEY,
  "customer_id" uuid,
  "country" varchar,
  "city" varchar,
  "address" varchar,
  "postal_code" varchar,
  "desc" varchar,
  "created_at" timestamp,
  "updated_at" timestamp
);

ALTER TABLE "products" ADD FOREIGN KEY ("store_id") REFERENCES "stores" ("id");

ALTER TABLE "products" ADD FOREIGN KEY ("category_id") REFERENCES "categories" ("id");

ALTER TABLE "products" ADD FOREIGN KEY ("brand_id") REFERENCES "brands" ("id");

ALTER TABLE "products" ADD FOREIGN KEY ("supplier_id") REFERENCES "suppliers" ("id");

ALTER TABLE "orders" ADD FOREIGN KEY ("supplier_id") REFERENCES "suppliers" ("id");

ALTER TABLE "orders" ADD FOREIGN KEY ("store_id") REFERENCES "stores" ("id");

ALTER TABLE "orders" ADD FOREIGN KEY ("created_by") REFERENCES "employees" ("id");

ALTER TABLE "order_products" ADD FOREIGN KEY ("product_id") REFERENCES "products" ("id");

ALTER TABLE "order_products" ADD FOREIGN KEY ("order_id") REFERENCES "orders" ("id");

ALTER TABLE "order_products" ADD FOREIGN KEY ("created_by") REFERENCES "employees" ("id");

ALTER TABLE "employees" ADD FOREIGN KEY ("client_type_id") REFERENCES "client_types" ("id");

ALTER TABLE "employees" ADD FOREIGN KEY ("role_id") REFERENCES "roles" ("id");

ALTER TABLE "customers" ADD FOREIGN KEY ("group_id") REFERENCES "groups" ("id");

ALTER TABLE "customers" ADD FOREIGN KEY ("tag_id") REFERENCES "tags" ("id");

ALTER TABLE "addresses" ADD FOREIGN KEY ("customer_id") REFERENCES "customers" ("id");
