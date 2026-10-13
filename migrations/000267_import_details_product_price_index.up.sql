-- Rezerv ro'yxatida top_selling=true har bir mahsulotning ENG OXIRGI importdagi
-- retail_price_vat narxini oladi (sotilgan pachka × narx bo'yicha saralash uchun).
-- Mavjud indeks (import_id, product_id) bunga yaramaydi: product_id yetakchi ustun emas,
-- shuning uchun product_id bo'yicha qidirishda butun jadval skanerlanardi.
--
-- INCLUDE (retail_price_vat) — index only scan: narx uchun jadvalga borish kerak emas.
--
-- DIQQAT (prod): import_details katta jadval, oddiy CREATE INDEX uni yozish uchun bloklaydi.
-- Prod'da avval qo'lda bajarish tavsiya etiladi:
--   CREATE INDEX CONCURRENTLY idx_import_details_product_created
--     ON import_details (product_id, created_at DESC) INCLUDE (retail_price_vat);
-- Shundan keyin bu migratsiya IF NOT EXISTS tufayli hech narsa qilmaydi.
CREATE INDEX IF NOT EXISTS idx_import_details_product_created
    ON import_details (product_id, created_at DESC) INCLUDE (retail_price_vat);
