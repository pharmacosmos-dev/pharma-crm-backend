ALTER TABLE "employee_attendance_days" DROP COLUMN IF EXISTS "comment";

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'employee_attendance_days' AND column_name = 'worked_minutes'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'employee_attendance_days' AND column_name = 'worker_minutes'
    ) THEN
        ALTER TABLE "employee_attendance_days" RENAME COLUMN "worked_minutes" TO "worker_minutes";
    END IF;
END $$;

ALTER TABLE "employees" DROP COLUMN IF EXISTS "end_date";
ALTER TABLE "employees" DROP COLUMN IF EXISTS "start_date";
