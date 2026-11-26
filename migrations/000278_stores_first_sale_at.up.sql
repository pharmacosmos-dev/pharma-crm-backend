-- Har bir do'konning birinchi yakunlangan savdosi vaqti
-- (sales.stage = 9, sale_type = 'SALE' va completed_at bor bo'lgan eng kichik
-- created_at). Keyingi savdolar uchun FinalizeSale / EposResult yangilaydi.
ALTER TABLE stores ADD COLUMN IF NOT EXISTS "first_sale_at" TIMESTAMP;

UPDATE stores st
SET first_sale_at = fs.first_sale_at
FROM (
    SELECT store_id, MIN(created_at) AS first_sale_at
    FROM sales
    WHERE stage = 9
      AND sale_type = 'SALE'
      AND completed_at IS NOT NULL
    GROUP BY store_id
) fs
WHERE fs.store_id = st.id;
