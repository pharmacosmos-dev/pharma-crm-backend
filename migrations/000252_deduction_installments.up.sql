-- Qarzni oylarga bo'lib to'lash.
--
-- deduction_details endi QARZNING O'ZI: pereuchyotdan chiqqan kamomad xodimga
-- qancha ajratilgani (masalan 4 mln) va necha oyga bo'linishi.
-- deduction_installments esa OYLIK JADVAL: har bir oyga alohida qator, o'z
-- yil/oyi, summasi va to'lov holati bilan.
--
-- Nega alohida jadval: qarz bitta oyga tegishli (pereuchyot o'sha oyda chiqdi),
-- to'lovlar esa keyingi oylarga tarqaladi. Ularni bitta qatorga sig'dirib
-- bo'lmaydi va oylik sarlavhalarga (deductions) tiqishtirsa, sarlavha summasi
-- o'sha oydagi pereuchyot summasi bo'lmay qolardi.

ALTER TABLE "deduction_details"
    ADD COLUMN IF NOT EXISTS "months_count" INTEGER NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS "deduction_installments" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    -- Qarz o'chirilsa jadval ham ketadi
    "deduction_detail_id" UUID NOT NULL REFERENCES "deduction_details" ("id") ON DELETE CASCADE,

    -- employee_id/store_id qarzda ham bor, lekin bu yerda takrorlanadi: "shu
    -- xodimning shu oydagi qarzi" so'rovi JOIN'siz o'qiladi — aynan shu so'rov
    -- har oy oylik ekranida ishlatiladi.
    "employee_id" UUID NOT NULL REFERENCES "employees" ("id"),
    "store_id"    UUID NOT NULL REFERENCES "stores" ("id"),

    -- Qaysi oyga tegishli to'lov
    "year"  INTEGER NOT NULL,
    "month" INTEGER NOT NULL,
    -- seq — nechanchi to'lov (1..months_count). Tartib va "oxirgi to'lov"ni
    -- aniqlash uchun: qoldiq tiyinlar oxirgisiga qo'shiladi.
    "seq"   INTEGER NOT NULL,

    "amount"  NUMERIC(20,2) NOT NULL DEFAULT 0,
    "is_paid" BOOLEAN       NOT NULL DEFAULT FALSE,
    "paid_at" TIMESTAMP WITH TIME ZONE,
    "comment" TEXT,

    "updated_by"  UUID REFERENCES "employees" ("id"),
    "approved_by" UUID REFERENCES "employees" ("id"),
    "approved_at" TIMESTAMP WITH TIME ZONE,

    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_deduction_installments_detail_seq UNIQUE ("deduction_detail_id", "seq")
);

-- Oylik ekranining asosiy so'rovi: "shu xodimning shu oydagi to'lanmagan qarzi"
CREATE INDEX IF NOT EXISTS idx_deduction_installments_employee_year_month
    ON "deduction_installments" ("employee_id", "year", "month");
CREATE INDEX IF NOT EXISTS idx_deduction_installments_store_year_month
    ON "deduction_installments" ("store_id", "year", "month");
CREATE INDEX IF NOT EXISTS idx_deduction_installments_detail
    ON "deduction_installments" ("deduction_detail_id");
CREATE INDEX IF NOT EXISTS idx_deduction_installments_unpaid
    ON "deduction_installments" ("is_paid") WHERE NOT "is_paid";
