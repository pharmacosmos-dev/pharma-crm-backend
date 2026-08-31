ALTER TABLE "employees"
    ADD COLUMN IF NOT EXISTS "daily_work_hours" NUMERIC(5,2) NOT NULL DEFAULT 0;

ALTER TABLE "employees"
    DROP COLUMN IF EXISTS "avg_monthly_hours";
