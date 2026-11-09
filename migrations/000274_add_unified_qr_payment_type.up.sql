ALTER TABLE "sales"
    ADD COLUMN IF NOT EXISTS "unified_qr" NUMERIC(10, 2) DEFAULT 0;

INSERT INTO "payment_types" ("name", "type", "description", "front_name", "is_active", "order_number")
SELECT 'Unified QR', 'card','Yagona QR orqali to''lov', 'unified_qr', true, 11
WHERE NOT EXISTS (SELECT 1 FROM "payment_types" WHERE "front_name" = 'unified_qr');


