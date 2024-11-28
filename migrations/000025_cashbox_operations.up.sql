CREATE TABLE IF NOT EXISTS cashbox_operations (
    id UUID NOT NULL PRIMARY KEY,
    cash_box_id UUID REFERENCES cash_boxes(id),
    employee_id UUID REFERENCES employees(id),
    cash_amount NUMERIC(10, 2),
    cashless_amount NUMERIC(10, 2),
    is_open BOOLEAN,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);