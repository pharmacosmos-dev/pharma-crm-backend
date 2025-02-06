CREATE TABLE IF NOT EXISTS requests_1c (
   "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "method" VARCHAR(255),
    "payload" JSONB,
    "response" JSONB,
    "action" VARCHAR(55),
    "created_at" TIMESTAMP DEFAULT NOW(),
    "updated_at" TIMESTAMP DEFAULT NOW()
);