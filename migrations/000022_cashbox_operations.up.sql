CREATE TABLE IF NOT EXISTS "cashbox_operations" (
    "id" UUID NOT NULL PRIMARY KEY,
    "cash_box_id" UUID REFERENCES cash_boxes(id),
    "cash_amount" NUMERIC(10, 2),
    "cashless_amount" NUMERIC(10, 2),
    "is_open" BOOLEAN,
    "description" VARCHAR(255),
    "start_time" TIMESTAMP,
    "end_time" TIMESTAMP,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);