ALTER TABLE "employees"
    ADD COLUMN IF NOT EXISTS "avg_monthly_hours" NUMERIC(10,2) NOT NULL DEFAULT 0;

ALTER TABLE "employees"
    DROP COLUMN IF EXISTS "daily_work_hours";
