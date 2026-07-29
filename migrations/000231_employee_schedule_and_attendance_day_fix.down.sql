ALTER TABLE "employee_attendance_days" DROP COLUMN IF EXISTS "comment";
ALTER TABLE "employee_attendance_days" RENAME COLUMN "worked_minutes" TO "worker_minutes";

ALTER TABLE "employees" DROP COLUMN IF EXISTS "end_date";
ALTER TABLE "employees" DROP COLUMN IF EXISTS "start_date";
