
ALTER TABLE "employees"
    ADD COLUMN IF NOT EXISTS "shift_type" VARCHAR(10);
