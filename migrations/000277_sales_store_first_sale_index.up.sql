-- Do'konlar hisobotida (GetStoreAmountReport) store_code / created_at bo'yicha
-- tartib har bir do'konning birinchi yakunlangan savdosini oladi:
--   MIN(created_at) WHERE store_id = ? AND stage = 9 AND sale_type = 'SALE'
-- Mavjud indekslar bunga yaramaydi: (store_id, stage) da created_at yo'q —
-- do'konning hamma savdosi o'qiladi; (created_at, store_id) da store_id
-- yetakchi emas. Bu indeks bilan har do'kon uchun bitta indeks qidiruvi.
--
-- DIQQAT (prod): sales katta jadval, oddiy CREATE INDEX uni yozish uchun bloklaydi.
-- Prod'da avval qo'lda bajarish tavsiya etiladi:
--   CREATE INDEX CONCURRENTLY idx_sales_store_first_sale
--     ON sales (store_id, created_at) WHERE stage = 9 AND sale_type = 'SALE';
-- Shundan keyin bu migratsiya IF NOT EXISTS tufayli hech narsa qilmaydi.
CREATE INDEX IF NOT EXISTS idx_sales_store_first_sale
    ON sales (store_id, created_at) WHERE stage = 9 AND sale_type = 'SALE';
