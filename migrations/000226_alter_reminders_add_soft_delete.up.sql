ALTER TABLE "reminders" ADD COLUMN IF NOT EXISTS "is_active" BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE "reminders" ADD COLUMN IF NOT EXISTS "deleted_at" TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_reminders_deleted_at ON reminders (deleted_at);
