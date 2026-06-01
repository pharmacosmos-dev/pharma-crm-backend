ALTER TABLE transfer_details
    ALTER COLUMN scanned_pack  DROP NOT NULL,
    ALTER COLUMN scanned_unit  DROP NOT NULL,
    ALTER COLUMN expected_pack DROP NOT NULL,
    ALTER COLUMN expected_unit DROP NOT NULL;
