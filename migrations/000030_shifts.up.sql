CREATE TABLE IF NOT EXISTS "shifts" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "from_employee_id" UUID REFERENCES employees(id),
    "to_employee_id" UUID REFERENCES employees(id),
    "cash_box_id" UUID REFERENCES cash_boxes(id),
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);