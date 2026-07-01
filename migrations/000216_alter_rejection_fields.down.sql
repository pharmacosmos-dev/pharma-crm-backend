ALTER TABLE transfer_details
    DROP COLUMN IF EXISTS "rejection_count",
    DROP COLUMN IF EXISTS "rejection_pack",
    DROP COLUMN IF EXISTS "rejection_unit";

ALTER TABLE transfers
    DROP COLUMN IF EXISTS "rejection_count",
    DROP COLUMN IF EXISTS "driver_rejection",
    DROP COLUMN IF EXISTS "rejection_by";
