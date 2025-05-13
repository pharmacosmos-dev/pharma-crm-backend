ALTER TABLE import_details ADD COLUMN IF NOT EXISTS "received_unit_count" INT DEFAULT 0;
ALTER TABLE import_details ADD COLUMN IF NOT EXISTS "scanned_unit_count" INT DEFAULT 0;
ALTER TABLE import_details ADD COLUMN IF NOT EXISTS "accepted_unit_count" INT DEFAULT 0;