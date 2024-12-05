CREATE TABLE IF NOT EXISTS "sales" (
    "id" UUID NOT NULL PRIMARY KEY,
    "employee_id" UUID REFERENCES employees(id),
    "cash_box_id" UUID REFERENCES cash_boxes(id),
    "sale_number" VARCHAR(10),
    "total_discount" NUMERIC(10, 2),
    "total_amount" NUMERIC(10, 2),
    "created_by" UUID,
    "updated_by" UUID,
    "deleted_by" UUID,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

