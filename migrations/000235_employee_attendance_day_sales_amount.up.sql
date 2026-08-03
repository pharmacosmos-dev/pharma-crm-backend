ALTER TABLE "employee_attendance_days" ADD COLUMN IF NOT EXISTS "sales_amount" NUMERIC(18,2) NOT NULL DEFAULT 0;
