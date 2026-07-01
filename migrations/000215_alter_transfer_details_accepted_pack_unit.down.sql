ALTER TABLE transfer_details
    DROP COLUMN IF EXISTS "accepted_pack",
    DROP COLUMN IF EXISTS "accepted_unit";
