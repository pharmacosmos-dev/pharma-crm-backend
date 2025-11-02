ALTER TABLE 
    "imports"
        DROP COLUMN IF EXISTS "received_count",
        DROP COLUMN IF EXISTS "received_sum",
        DROP COLUMN IF EXISTS "scanned_count",
        DROP COLUMN IF EXISTS "scanned_sum";