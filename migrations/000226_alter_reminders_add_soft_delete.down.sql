DROP INDEX IF EXISTS idx_reminders_deleted_at;

ALTER TABLE "reminders" DROP COLUMN IF EXISTS "deleted_at";
ALTER TABLE "reminders" DROP COLUMN IF EXISTS "is_active";
