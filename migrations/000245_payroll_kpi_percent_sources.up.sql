ALTER TABLE "employee_payrolls"
    ADD COLUMN IF NOT EXISTS "plan_kpi_percent" NUMERIC(5,2) NOT NULL DEFAULT 0;

ALTER TABLE "employee_payrolls"
    ADD COLUMN IF NOT EXISTS "employee_kpi_percent" NUMERIC(5,2) NOT NULL DEFAULT 0;
