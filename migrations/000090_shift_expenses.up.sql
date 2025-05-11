CREATE TABLE IF NOT EXISTS shift_expenses(
    "id" SERIAL PRIMARY KEY,
    "store_id" UUID REFERENCES stores(id),
    "cashbox_id" UUID REFERENCES cash_boxes(id),
    "cashbox_operation_id" UUID REFERENCES cashbox_operations(id),
    "docs_number" VARCHAR(55) NOT NULL UNIQUE,
    "total_quantity" NUMERIC(10, 2) DEFAULT 0,
    "total_amount" NUMERIC(18, 2) DEFAULT 0,
    "status" SMALLINT DEFAULT 0,
    "sent_at" TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    "created_at" TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);