ALTER TABLE "employees" ADD COLUMN IF NOT EXISTS "start_date" TIME;
ALTER TABLE "employees" ADD COLUMN IF NOT EXISTS "end_date" TIME;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'employee_attendance_days' AND column_name = 'worker_minutes'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'employee_attendance_days' AND column_name = 'worked_minutes'
    ) THEN
        ALTER TABLE "employee_attendance_days" RENAME COLUMN "worker_minutes" TO "worked_minutes";
    END IF;
END $$;

ALTER TABLE "employee_attendance_days" ADD COLUMN IF NOT EXISTS "comment" TEXT;
ALTER TABLE "employee_attendance_days" ALTER COLUMN "planned_start_at" DROP NOT NULL;
