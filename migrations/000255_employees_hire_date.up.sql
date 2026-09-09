-- hire_date — xodim ishga qabul qilingan SANA.
--
-- DIQQAT: mavjud employees.start_date bilan adashtirmang — u TIME turida va
-- xodimning smenasi boshlanish VAQTI (davomat hisobida ishlatiladi).
--
-- Nullable: eski xodimlarda bu sana ma'lum emas va majburiy qilib bo'lmaydi.
ALTER TABLE "employees" ADD COLUMN IF NOT EXISTS "hire_date" DATE;
