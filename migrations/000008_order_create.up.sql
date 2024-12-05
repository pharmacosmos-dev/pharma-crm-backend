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
  "created_by" uuid,
  "updated_by" uuid,
  "deleted_by" uuid,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);