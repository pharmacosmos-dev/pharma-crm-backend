DELETE FROM "cashbox_payment_types"
WHERE "payment_type_id" IN (SELECT id FROM "payment_types" WHERE "front_name" = 'unified_qr');

DELETE FROM "payment_types" WHERE "front_name" = 'unified_qr';

ALTER TABLE "sales"
    DROP COLUMN IF EXISTS "unified_qr";
