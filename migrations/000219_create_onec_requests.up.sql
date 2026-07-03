CREATE TABLE IF NOT EXISTS "onec_requests" (
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

CREATE INDEX IF NOT EXISTS idx_onec_requests_method ON onec_requests(method);
CREATE INDEX IF NOT EXISTS idx_onec_requests_created_at ON onec_requests(created_at);
