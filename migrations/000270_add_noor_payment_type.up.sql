ALTER TABLE "sales"
    ADD COLUMN IF NOT EXISTS "noor" NUMERIC(10, 2) DEFAULT 0;

INSERT INTO "payment_types" ("name", "type", "description", "front_name", "is_active", "order_number")
SELECT 'Noor', 'online_order', 'Noor orqali to''lov', 'noor', true, 10
WHERE NOT EXISTS (SELECT 1 FROM "payment_types" WHERE "front_name" = 'noor');

-- noor orders were created with the uzumtezkor payment type; unfinished ones must be closed with noor
UPDATE "sales"
SET "payment_type" = 'noor'
WHERE "service_type" = 'noor'
  AND "type" = 'online'
  AND "payment_type" = 'uzumtezkor'
  AND "stage" NOT IN (9, 11);
