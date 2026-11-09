DELETE FROM "cashbox_payment_types"
WHERE "payment_type_id" IN (SELECT id FROM "payment_types" WHERE "front_name" IN ('unified_qr', 'Yagona QR'));

DELETE FROM "payment_types" WHERE "front_name" IN ('unified_qr', 'Yagona QR');

ALTER TABLE "sales"
    DROP COLUMN IF EXISTS "unified_qr";
