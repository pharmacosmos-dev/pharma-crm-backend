CREATE TABLE IF NOT EXISTS "dmed_requests" (
    "id"       BIGSERIAL      PRIMARY KEY,
    "payload"  JSONB          NOT NULL,
    "method"   VARCHAR(10)    NOT NULL,
    "response" JSONB          NULL,
    "status"   SMALLINT       NOT NULL DEFAULT 0, -- 0 -> failed, 1 -> success
    "created_at" TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);