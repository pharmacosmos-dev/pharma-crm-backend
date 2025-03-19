CREATE TABLE IF NOT EXISTS "app_transactions" (
    "id" SERIAL PRIMARY KEY,
    "sale_id" UUID REFERENCES sales(id),
    "payment_id" BIGINT,
    "payment_status" SMALLINT,
    "receipt_id" VARCHAR(255),
    "created_at" TIMESTAMP DEFAULT NOW(),
    "updated_at" TIMESTAMP DEFAULT NOW()
);