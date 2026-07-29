ALTER TABLE "employees" ADD COLUMN IF NOT EXISTS "start_date" TIME;
ALTER TABLE "employees" ADD COLUMN IF NOT EXISTS "end_date" TIME;

ALTER TABLE "employee_attendance_days" RENAME COLUMN "worker_minutes" TO "worked_minutes";
ALTER TABLE "employee_attendance_days" ADD COLUMN IF NOT EXISTS "comment" TEXT;
ALTER TABLE "employee_attendance_days" ALTER COLUMN "planned_start_at" DROP NOT NULL;
