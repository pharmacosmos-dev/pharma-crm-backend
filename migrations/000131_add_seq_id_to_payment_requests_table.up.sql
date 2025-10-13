ALTER TABLE 
    "payment_requests"
        ADD COLUMN IF NOT EXISTS "seq_id" BIGSERIAL NOT NULL;