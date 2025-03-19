CREATE TABLE IF NOT EXISTS "payment_types" (
    "id" UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "name" VARCHAR(255),
    "type" VARCHAR(10),
    "description" TEXT,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "payment_services" (
    "id" UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "store_id" UUID REFERENCES "stores"("id") ON DELETE CASCADE,
    "name" VARCHAR(255),
    "type" VARCHAR(10),
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
    "id" UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "sale_id" UUID REFERENCES "sales"("id") ON DELETE CASCADE,
    "payment_service_id" UUID REFERENCES "payment_services"("id") ON DELETE CASCADE,
    "payment_type_id" UUID REFERENCES "payment_types"("id") ON DELETE CASCADE,
    "cash_box_operation_id" UUID REFERENCES "cashbox_operations"("id") ON DELETE CASCADE,
    "amount" NUMERIC(10, 2),
    "paid_at" TIMESTAMP,
    "status" VARCHAR(20),
    "transaction_id" UUID,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);