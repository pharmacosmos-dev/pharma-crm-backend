CREATE TABLE IF NOT EXISTS "attendance_logs" (
    "id"          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "store_id"    UUID REFERENCES "stores" ("id"),
    "employee_id" UUID REFERENCES "employees" ("id"),
    "event_type"  TEXT NOT NULL,
    "event_at"    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "raw_payload" JSONB,
    "face_id"     TEXT,
    "created_at"  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at"  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attendance_logs_employee_event_at ON attendance_logs (employee_id, event_at);
CREATE INDEX IF NOT EXISTS idx_attendance_logs_store_id ON attendance_logs (store_id);

