CREATE TABLE IF NOT EXISTS employee_bonus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID REFERENCES employees(id),
    sale_id UUID REFERENCES sales(id),
    cashbox_operation_id UUID REFERENCES cashbox_operations(id),
    bonus_amount NUMERIC(10,2) DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);