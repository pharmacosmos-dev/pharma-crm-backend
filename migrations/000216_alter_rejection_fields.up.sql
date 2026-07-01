ALTER TABLE transfer_details
    ADD COLUMN IF NOT EXISTS "rejection_count" FLOAT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "rejection_pack"  INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "rejection_unit"  INTEGER DEFAULT 0;

ALTER TABLE transfers
    ADD COLUMN IF NOT EXISTS "rejection_count"    FLOAT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "driver_rejection"   TEXT,
    ADD COLUMN IF NOT EXISTS "rejection_by"       UUID REFERENCES employees("id");
