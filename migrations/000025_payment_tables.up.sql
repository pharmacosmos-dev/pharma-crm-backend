CREATE TABLE IF NOT EXISTS "payment_types" (
    "id" UUID NOT NULL PRIMARY KEY,
    "name" VARCHAR(255),
    "type" VARCHAR(10),
    "description" TEXT,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "payment_services" (
    "id" UUID NOT NULL PRIMARY KEY,
    "cash_box_id" UUID REFERENCES "cash_boxes"("id"),
    "name" VARCHAR(255),
    "merchant_id" INT,
    "service_id" INT,
    "merchant_user_id" INT,
    "secret_key" VARCHAR(255),
    "is_active" BOOLEAN,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);

CREATE TABLE IF NOT EXISTS "sale_payments" (
    "id" UUID NOT NULL PRIMARY KEY,
    "sale_id" UUID REFERENCES "sales"("id"),
    "payment_service_id" UUID REFERENCES "payment_services"("id"),
    "payment_type_id" UUID REFERENCES "payment_types"("id"),
    "cash_box_id" UUID REFERENCES "cash_boxes"("id"),
    "cash_box_operation_id" UUID REFERENCES "cashbox_operations"("id"),
    "amount" NUMERIC(10, 2),
    "net_amount" NUMERIC(10, 2) DEFAULT 0,
    "expense_amount" NUMERIC(10, 2) DEFAULT 0,
    "paid_at" TIMESTAMP,
    "status" VARCHAR(20),
    "cash_box_status" VARCHAR(20) DEFAULT 'open',
    "transaction_id" UUID,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "transactions" (
    "id" UUID NOT NULL PRIMARY KEY,
    "sale_payment_id" UUID REFERENCES "sale_payments"("id"),
    "payment_service_id" UUID REFERENCES "payment_services"("id"),
    "transaction_id" UUID,
    "status" VARCHAR(20),
    "response_data" JSONB,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);