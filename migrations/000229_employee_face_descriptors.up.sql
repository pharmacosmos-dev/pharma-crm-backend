CREATE TABLE IF NOT EXISTS "employee_face_descriptors" (
    "id"          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "employee_id" UUID NOT NULL REFERENCES "employees" ("id") ON DELETE CASCADE,
    "descriptor"  JSONB NOT NULL,
    "created_at"  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at"  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_employee_face_descriptors_employee_id
    ON employee_face_descriptors (employee_id);
