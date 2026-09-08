-- Oylikdan ushlab qolishlar: shtraf, pereuchyot va boshqalar.
--
-- Uchta jadval:
--   deduction_types   — turlar lug'ati, admin yangi tur qo'sha oladi
--   deductions        — do'kon + oy sarlavhasi: to'liq to'landimi yoki ochiqmi
--   deduction_details — xodim bo'yicha aniq summalar
--
-- Hozircha employee_payrolls'dan MUSTAQIL: oylikdagi deduction_* ustunlari
-- avvalgidek qo'lda kiritiladi. Keyinchalik cron shu jadvallardan yig'adigan
-- qilinsa, o'sha yerda ulanadi.

CREATE TABLE IF NOT EXISTS "deduction_types" (
    "id"         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    -- code — kodda ishlatiladigan barqaror kalit. name tahrirlansa ham
    -- o'zgarmaydi, shuning uchun hisobotlar buzilmaydi.
    "code"       VARCHAR(50)  NOT NULL UNIQUE,
    "name"       VARCHAR(255) NOT NULL,
    "is_active"  BOOLEAN      NOT NULL DEFAULT TRUE,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- employee_payrolls'dagi uchta mavjud ustunga mos turlar
INSERT INTO "deduction_types" ("code", "name") VALUES
    ('TERM',    'Срок'),
    ('RECOUNT', 'Переучёт'),
    ('FINE',    'Штраф')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS "deductions" (
    "id"         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "store_id"   UUID NOT NULL REFERENCES "stores" ("id"),
    "company_id" UUID,
    "year"       INTEGER NOT NULL,
    "month"      INTEGER NOT NULL,

    -- total_amount — detallar yig'indisi, paid_amount — shundan to'langani.
    -- Ikkalasi saqlanadi: "qancha qoldi" har safar detallarni yig'masdan
    -- ko'rinadi va yopilgan oyning summasi keyin o'zgarmaydi.
    "total_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
    "paid_amount"  NUMERIC(20,2) NOT NULL DEFAULT 0,

    -- 'open'  — hali to'liq yopilmagan
    -- 'paid'  — to'liq to'langan
    "status"     VARCHAR(20) NOT NULL DEFAULT 'open',
    "comment"    TEXT,

    "created_by"  UUID REFERENCES "employees" ("id"),
    "updated_by"  UUID REFERENCES "employees" ("id"),
    "approved_by" UUID REFERENCES "employees" ("id"),
    "approved_at" TIMESTAMP WITH TIME ZONE,
    -- completed_at — to'liq to'langan payt
    "completed_at" TIMESTAMP WITH TIME ZONE,

    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Bir do'konga bir oyda bitta sarlavha
    CONSTRAINT uq_deductions_store_year_month UNIQUE ("store_id", "year", "month")
);

CREATE INDEX IF NOT EXISTS idx_deductions_company_year_month ON "deductions" ("company_id", "year", "month");
CREATE INDEX IF NOT EXISTS idx_deductions_status ON "deductions" ("status");

CREATE TABLE IF NOT EXISTS "deduction_details" (
    "id"                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    -- Sarlavha o'chirilsa uning qatorlari ham ketadi
    "deduction_id"      UUID NOT NULL REFERENCES "deductions" ("id") ON DELETE CASCADE,
    "deduction_type_id" UUID NOT NULL REFERENCES "deduction_types" ("id"),
    "employee_id"       UUID NOT NULL REFERENCES "employees" ("id"),

    -- store_id/year/month sarlavhada ham bor, lekin bu yerda takrorlanadi:
    -- xodim bo'yicha hisobot sarlavhaga JOIN qilmasdan o'qiladi.
    "store_id"          UUID NOT NULL REFERENCES "stores" ("id"),
    "year"              INTEGER NOT NULL,
    "month"             INTEGER NOT NULL,

    "amount"  NUMERIC(20,2) NOT NULL DEFAULT 0,
    "is_paid" BOOLEAN       NOT NULL DEFAULT FALSE,
    "paid_at" TIMESTAMP WITH TIME ZONE,
    "comment" TEXT,

    "created_by"  UUID REFERENCES "employees" ("id"),
    "updated_by"  UUID REFERENCES "employees" ("id"),
    "approved_by" UUID REFERENCES "employees" ("id"),
    "approved_at" TIMESTAMP WITH TIME ZONE,

    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- UNIQUE ataylab YO'Q: bitta xodimga bir oyda bir necha shtraf yozilishi mumkin,
-- har biri alohida qator bo'lib qoladi.
CREATE INDEX IF NOT EXISTS idx_deduction_details_deduction ON "deduction_details" ("deduction_id");
CREATE INDEX IF NOT EXISTS idx_deduction_details_employee_year_month ON "deduction_details" ("employee_id", "year", "month");
CREATE INDEX IF NOT EXISTS idx_deduction_details_store_year_month ON "deduction_details" ("store_id", "year", "month");
CREATE INDEX IF NOT EXISTS idx_deduction_details_type ON "deduction_details" ("deduction_type_id");
