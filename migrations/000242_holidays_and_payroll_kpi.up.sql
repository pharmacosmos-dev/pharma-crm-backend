-- Dam olish kunlari. Ish kunlari hisobida: kalendar kun − yakshanba − shu jadvaldagi sana.
-- Yakshanba kodda aniqlanadi (EXTRACT(DOW) = 0), bu yerda faqat bayramlar turadi.
-- Hayit sanalari har yili o'zgaradi — ularni admin har yili qo'shib boradi.
CREATE TABLE IF NOT EXISTS "holidays" (
    "id"         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "date"       DATE NOT NULL UNIQUE,
    "name"       VARCHAR(255) NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_holidays_date ON holidays (date);

-- Har yili takrorlanadigan qat'iy sanalar 2025-2035 uchun oldindan to'ldiriladi.
-- Hayitlar bu yerda yo'q: sanasi oy kalendariga bog'liq, qo'lda kiritiladi.
INSERT INTO holidays (date, name)
SELECT make_date(y, 12, 31), 'Yangi yil bayrami' FROM generate_series(2025, 2035) AS y
ON CONFLICT (date) DO NOTHING;

INSERT INTO holidays (date, name)
SELECT make_date(y, 3, 21), 'Navro''z bayrami' FROM generate_series(2025, 2035) AS y
ON CONFLICT (date) DO NOTHING;

-- Xodim va do'kon nomlari endi payroll qatoriga snapshot qilinadi: hisobot
-- employee_payrolls'dan JOIN'siz o'qiladi, xodim keyin ko'chirilsa ham eski oy
-- hisoboti o'sha paytdagi holatni ko'rsatadi.
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "company_id" UUID;
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "first_name" VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "last_name"  VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "full_name"  VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "store_name" VARCHAR(255) NOT NULL DEFAULT '';

-- role       — ko'rsatish uchun ("Кассир" yoki "Заведующий, Кассир")
-- role_names — filtrlash uchun; xodimda bir nechta rol bo'lgani sabab massiv,
--              `role_names && ARRAY[...]` bilan aniq va indeksli qidiriladi
--              (satr ichidan LIKE bilan qidirish noto'g'ri natija berardi).
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "role"       VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "role_names" TEXT[] NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_employee_payrolls_company_year_month ON employee_payrolls (company_id, year, month);
CREATE INDEX IF NOT EXISTS idx_employee_payrolls_role_names ON employee_payrolls USING GIN (role_names);

-- KPI progressiv hisoblanadi, quyidagilar uni shaffof qiladi:
--   employee_plan_amount     — xodimning to'liq oylik rejasi (employee_targets.amount)
--   expected_plan_amount     — o'tgan ish kunlariga proporsional kesilgan reja
--   plan_achievement_percent — individual_sales / expected_plan * 100
--   month_work_days          — oydagi jami ish kunlari
--   elapsed_work_days        — oy boshidan hisob sanasigacha o'tgan ish kunlari
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "employee_plan_amount"     NUMERIC(20,2) NOT NULL DEFAULT 0;
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "expected_plan_amount"     NUMERIC(20,2) NOT NULL DEFAULT 0;
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "plan_achievement_percent" NUMERIC(8,2)  NOT NULL DEFAULT 0;
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "month_work_days"          INTEGER       NOT NULL DEFAULT 0;
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "elapsed_work_days"        INTEGER       NOT NULL DEFAULT 0;

-- Cron oxirgi marta qachon qayta hisoblagani (kunlik yangilanish nazorati uchun).
ALTER TABLE "employee_payrolls" ADD COLUMN IF NOT EXISTS "calculated_at" TIMESTAMP WITH TIME ZONE;

-- Cron kunlik UPSERT qiladi: ON CONFLICT (employee_id, year, month) DO UPDATE.
-- Konstrukta 000239'da yaratilgan (uq_employee_payrolls_employee_year_month).
