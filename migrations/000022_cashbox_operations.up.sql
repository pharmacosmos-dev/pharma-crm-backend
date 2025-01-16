CREATE TABLE IF NOT EXISTS "cashbox_operations" (
    "id" UUID NOT NULL PRIMARY KEY,
    "cash_box_id" UUID REFERENCES cash_boxes(id),
    "cash_amount" NUMERIC(10, 2) DEFAULT 0,
    "cashless_amount" NUMERIC(10, 2) DEFAULT 0,
    "opened_amount" NUMERIC(10, 2) DEFAULT 0,
    "closed_amount" NUMERIC(10, 2) DEFAULT 0,
    "is_open" BOOLEAN,
    "description" VARCHAR(255),
    "start_time" TIMESTAMP,
    "end_time" TIMESTAMP,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX unique_open_cash_per_employee
ON "cashbox_operations" ("cash_box_id", "employee_id")
WHERE "end_time" IS NULL;