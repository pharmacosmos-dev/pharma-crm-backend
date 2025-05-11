CREATE TABLE IF NOT EXISTS cashbox_closures(
    "id" SERIAL PRIMARY KEY,
    "cashbox_operation_id" UUID REFERENCES cashbox_operations(id) ON DELETE CASCADE,
    "sender_id" UUID REFERENCES employees(id),
    "receiver_id" UUID REFERENCES employees(id),
    "received_amount" NUMERIC(20, 2) DEFAULT 0.00,
    "accepted_amount" NUMERIC(20, 2) DEFAULT 0.00,
    "status" VARCHAR(55) DEFAULT 'pending',
    "comment" TEXT,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);