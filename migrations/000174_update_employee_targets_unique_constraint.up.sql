ALTER TABLE employee_targets
    DROP CONSTRAINT IF EXISTS employee_targets_employee_id_year_month_key;

ALTER TABLE employee_targets
    ADD CONSTRAINT employee_targets_employee_id_store_id_year_month_key
    UNIQUE (employee_id, store_id, year, month);
