
CREATE INDEX IF NOT EXISTS idx_employee_bonus_employee_created
    ON employee_bonus (employee_id, created_at);
