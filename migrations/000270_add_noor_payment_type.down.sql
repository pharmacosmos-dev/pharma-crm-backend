UPDATE "sales"
SET "payment_type" = 'uzumtezkor'
WHERE "service_type" = 'noor'
  AND "type" = 'online'
  AND "payment_type" = 'noor';

DELETE FROM "payment_types" WHERE "front_name" = 'noor';

ALTER TABLE "sales"
    DROP COLUMN IF EXISTS "noor";
