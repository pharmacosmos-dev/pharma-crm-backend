-- Attendance yozuvini kim yaratgani / kim tuzatgani.
-- created_by = JWT tokendagi user_id: face-id orqali check-in qilganda xodimning
-- o'zi (created_by = employee_id), qo'lda kiritilganda esa kiritgan admin.
-- created_by IS NULL — cron (auto-close) yoki to'g'ridan-to'g'ri SQL orqali yozilgan.
-- updated_by — event_at'ni oxirgi marta qo'lda tuzatgan foydalanuvchi (cron tuzatsa NULL qoladi).
ALTER TABLE "attendance_logs"
    ADD COLUMN IF NOT EXISTS "created_by" UUID REFERENCES "employees" ("id") ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS "updated_by" UUID REFERENCES "employees" ("id") ON DELETE SET NULL;
