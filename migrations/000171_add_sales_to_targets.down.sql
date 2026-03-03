ALTER TABLE store_targets
    DROP COLUMN IF EXISTS sales;

ALTER TABLE employee_targets
    DROP COLUMN IF EXISTS sales;
