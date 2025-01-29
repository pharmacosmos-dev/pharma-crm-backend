CREATE TABLE IF NOT EXISTS "cashbox_operations" (
    "id" UUID NOT NULL PRIMARY KEY,
    "cash_box_id" UUID REFERENCES cash_boxes(id),
    "employee_id" UUID REFERENCES employees(id),
    "current_employee_id" UUID REFERENCES employees(id),
    "cash_amount" NUMERIC(10, 2) DEFAULT 0,
    "cashless_amount" NUMERIC(10, 2) DEFAULT 0,
    "opened_amount" NUMERIC(10, 2) DEFAULT 0,
    "closed_amount" NUMERIC(10, 2) DEFAULT 0,
    "is_open" BOOLEAN,
    "description" VARCHAR(255),
    "start_time" TIMESTAMP,
    "end_time" TIMESTAMP,
    "cash" NUMERIC(10, 2) DEFAULT 0,
    "uzcard" NUMERIC(10, 2) DEFAULT 0,
    "humo" NUMERIC(10, 2) DEFAULT 0,
    "visa" NUMERIC(10, 2) DEFAULT 0,
    "click" NUMERIC(10, 2) DEFAULT 0,
    "payme" NUMERIC(10, 2) DEFAULT 0,
    "uzum" NUMERIC(10, 2) DEFAULT 0,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS unique_open_cash_per_employee
ON "cashbox_operations" ("cash_box_id", "employee_id")
WHERE "end_time" IS NULL;