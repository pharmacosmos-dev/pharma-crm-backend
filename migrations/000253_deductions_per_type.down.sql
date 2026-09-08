ALTER TABLE "deductions" DROP CONSTRAINT IF EXISTS "uq_deductions_store_year_month_type";
ALTER TABLE "deductions"
    ADD CONSTRAINT "uq_deductions_store_year_month" UNIQUE ("store_id", "year", "month");
ALTER TABLE "deductions" DROP COLUMN IF EXISTS "deduction_type_id";
