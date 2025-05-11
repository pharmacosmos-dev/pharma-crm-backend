CREATE TABLE IF NOT EXISTS finance_operations(
    "id" SERIAL PRIMARY KEY,
    "cashbox_id" UUID REFERENCES cash_boxes(id),
    "employee_id" UUID REFERENCES employees(id),
    "amount" NUMERIC(20, 2) DEFAULT 0.00,
    "operation_type" VARCHAR(55),
    "comment" TEXT,
    "status" VARCHAR(55),
    "report_type" VARCHAR(55),
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);