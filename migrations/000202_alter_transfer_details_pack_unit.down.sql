ALTER TABLE transfer_details
    DROP COLUMN IF EXISTS "expected_pack",
    DROP COLUMN IF EXISTS "expected_unit";
