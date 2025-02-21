CREATE TABLE IF NOT EXISTS "epos_responses" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "sale_id" UUID REFERENCES sales(id) ON DELETE CASCADE,
    "response" JSONB,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
)