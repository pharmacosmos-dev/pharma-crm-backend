CREATE TABLE IF NOT EXISTS "uzum_order_logs" (
    "id"          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "method"      VARCHAR(200) NOT NULL,
    "payload"     JSONB,
    "response"    JSONB,
    "status_code" INTEGER,
    "duration_ms" INTEGER,
    "token"       TEXT,
    "ip_address"  VARCHAR(50),
    "created_at"  TIMESTAMP DEFAULT NOW(),
    "updated_at"  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_uzum_order_logs_created_at ON uzum_order_logs(created_at);
