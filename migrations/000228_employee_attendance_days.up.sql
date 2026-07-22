CREATE TABLE IF NOT EXISTS "employee_attendance_days" (
    "id"                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "store_id"           UUID REFERENCES "stores" ("id"),
    "employee_id"        UUID REFERENCES "employees" ("id"),
    "work_date"          DATE NOT NULL,
    "planned_start_at"   TIMESTAMP WITH TIME ZONE NOT NULL,
    "first_check_in"     TIMESTAMP WITH TIME ZONE,
    "last_check_out"     TIMESTAMP WITH TIME ZONE,
    "worker_minutes"     INT NOT NULL DEFAULT 0,
    "late_minutes"       INT NOT NULL DEFAULT 0,
    "is_absent"          BOOLEAN NOT NULL DEFAULT FALSE,
    "is_manual_override" BOOLEAN NOT NULL DEFAULT FALSE,
    "updated_by"         UUID REFERENCES "employees" ("id"),
    "created_at"         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at"         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_employee_attendance_days_employee_date UNIQUE ("employee_id", "work_date")
);

CREATE INDEX IF NOT EXISTS idx_employee_attendance_days_store_work_date ON employee_attendance_days (store_id, work_date);

