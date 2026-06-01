ALTER TABLE transfer_details
    DROP COLUMN IF EXISTS "scanned_pack",
    DROP COLUMN IF EXISTS "scanned_unit";
