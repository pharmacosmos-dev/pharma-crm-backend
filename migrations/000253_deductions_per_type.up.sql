-- Sarlavha endi TUR bo'yicha ham ajratiladi: (do'kon, yil, oy, tur).
--
-- Ilgari bitta sarlavhada shtraf ham, pereuchyot ham turardi. Natijada bir
-- oyda to'liq to'langan shtraf, o'sha oydagi 7 oylik pereuchyot qarzi sabab
-- sarlavhani "ochiq" holatda ushlab turardi — status ma'nosini yo'qotgandi.
--
-- Endi har bir tur o'z sarlavhasiga ega: shtraf o'z oyida yopiladi, pereuchyot
-- esa alohida ochiq qoladi.

ALTER TABLE "deductions"
    ADD COLUMN IF NOT EXISTS "deduction_type_id" UUID REFERENCES "deduction_types" ("id");

-- Mavjud sarlavhalarga tur qo'yamiz: birinchi detalining turi bo'yicha.
-- Bu jadvallar endigina yaratilgani uchun amalda bo'sh, lekin migratsiya
-- ma'lumot bilan ham ishlashi kerak.
UPDATE "deductions" d
SET "deduction_type_id" = (
    SELECT dd."deduction_type_id"
    FROM "deduction_details" dd
    WHERE dd."deduction_id" = d."id"
    ORDER BY dd."created_at"
    LIMIT 1
)
WHERE d."deduction_type_id" IS NULL;

-- Detali yo'q sarlavhalar RECOUNT deb belgilanadi (eng ko'p uchraydigan tur)
UPDATE "deductions"
SET "deduction_type_id" = (SELECT "id" FROM "deduction_types" WHERE "code" = 'RECOUNT')
WHERE "deduction_type_id" IS NULL;

ALTER TABLE "deductions" ALTER COLUMN "deduction_type_id" SET NOT NULL;

-- Eski kalit: bir do'konga bir oyda bitta sarlavha
ALTER TABLE "deductions" DROP CONSTRAINT IF EXISTS "uq_deductions_store_year_month";
-- Yangi kalit: har bir tur uchun alohida
ALTER TABLE "deductions"
    ADD CONSTRAINT "uq_deductions_store_year_month_type"
    UNIQUE ("store_id", "year", "month", "deduction_type_id");

CREATE INDEX IF NOT EXISTS idx_deductions_type ON "deductions" ("deduction_type_id");
