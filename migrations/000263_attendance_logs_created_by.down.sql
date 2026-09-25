ALTER TABLE "attendance_logs"
    DROP COLUMN IF EXISTS "created_by",
    DROP COLUMN IF EXISTS "updated_by";
