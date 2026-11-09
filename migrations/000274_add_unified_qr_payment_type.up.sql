ALTER TABLE "sales"
    ADD COLUMN IF NOT EXISTS "unified_qr" NUMERIC(10, 2) DEFAULT 0;

INSERT INTO "payment_types" ("name", "type", "description", "front_name", "is_active", "order_number")
SELECT 'Yagona QR', 'card', 'Yagona QR orqali to''lov', 'Yagona QR', true, 11
WHERE NOT EXISTS (SELECT 1 FROM "payment_types" WHERE "front_name" IN ('unified_qr', 'Yagona QR'));


