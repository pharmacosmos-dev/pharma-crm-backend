-- KPI endi do'kon bo'yicha hisoblanadi: reja ham, savdo ham do'konniki.
-- store_sales_amount — do'konning oy boshidan calc_date'gacha bo'lgan savdosi
-- (sales jadvalidan). Qator xodimniki, lekin qiymat do'konga tegishli, ya'ni
-- bitta do'konning barcha xodimlarida bir xil bo'ladi.
ALTER TABLE "employee_payrolls"
    ADD COLUMN IF NOT EXISTS "store_sales_amount" NUMERIC(20,2) NOT NULL DEFAULT 0;

-- Payroll hisobi sales'ni sana oralig'i bo'yicha do'konlarga guruhlab o'qiydi.
-- Mavjud indekslar (store_id, stage) va (completed_at, stage, store_id) bunga
-- yaramaydi — filtr created_at bo'yicha. Partial: hisobga faqat yakunlangan
-- sotuvlar kiradi.
CREATE INDEX IF NOT EXISTS idx_sales_created_at_store
    ON sales (created_at, store_id)
    WHERE stage = 9 AND sale_type = 'SALE';
