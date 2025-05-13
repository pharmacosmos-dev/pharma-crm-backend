ALTER TABLE import_details ADD COLUMN IF NOT EXISTS "received_unit_count" NUMERIC(10, 4) DEFAULT 0.00;
ALTER TABLE import_details ADD COLUMN IF NOT EXISTS "scanned_unit_count" NUMERIC(10, 4) DEFAULT 0.00;
ALTER TABLE import_details ADD COLUMN IF NOT EXISTS "accepted_unit_count" NUMERIC(10, 4) DEFAULT 0.00;
ALTER TABLE import_details ALTER COLUMN "received_count" TYPE NUMERIC(10, 4);
ALTER TABLE import_details ALTER COLUMN "scanned_count" TYPE NUMERIC(10, 4);
ALTER TABLE import_details ALTER COLUMN "accepted_count" TYPE NUMERIC(10, 4);