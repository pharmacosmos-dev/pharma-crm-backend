UPDATE transfer_details SET scanned_pack = 0 WHERE scanned_pack IS NULL;
UPDATE transfer_details SET scanned_unit = 0 WHERE scanned_unit IS NULL;
UPDATE transfer_details SET expected_pack = 0 WHERE expected_pack IS NULL;
UPDATE transfer_details SET expected_unit = 0 WHERE expected_unit IS NULL;

ALTER TABLE transfer_details
    ALTER COLUMN scanned_pack  SET DEFAULT 0,
    ALTER COLUMN scanned_pack  SET NOT NULL,
    ALTER COLUMN scanned_unit  SET DEFAULT 0,
    ALTER COLUMN scanned_unit  SET NOT NULL,
    ALTER COLUMN expected_pack SET DEFAULT 0,
    ALTER COLUMN expected_pack SET NOT NULL,
    ALTER COLUMN expected_unit SET DEFAULT 0,
    ALTER COLUMN expected_unit SET NOT NULL;
