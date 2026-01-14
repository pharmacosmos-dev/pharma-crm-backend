ALTER TABLE
    "sales"
        ADD COLUMN IF NOT EXISTS "client_comment" TEXT DEFAULT NULL;