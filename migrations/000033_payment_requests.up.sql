CREATE TABLE IF NOT EXISTS "payment_requests" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "method" VARCHAR(255),
    "payload" JSONB,
    "response" JSONB,
    "payment_provider" VARCHAR(55),
    "transaction_id" UUID,
    "created_at" TIMESTAMP DEFAULT NOW(),
    "updated_at" TIMESTAMP DEFAULT NOW()
);