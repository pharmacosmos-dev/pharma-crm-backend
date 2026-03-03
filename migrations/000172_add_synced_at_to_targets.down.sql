ALTER TABLE store_targets
    DROP COLUMN IF EXISTS synced_at;

ALTER TABLE employee_targets
    DROP COLUMN IF EXISTS synced_at;
