CREATE TABLE IF NOT EXISTS cash_box_sessions (
    id UUID NOT NULL PRIMARY KEY,
    cash_box_id UUID REFERENCES cash_boxes(id),
    employee_id UUID REFERENCES employees(id),
    store_id UUID REFERENCES stores(id),
    type VARCHAR(50) CHECK (type IN ('with_cash', 'without_cash')),
    opening_balance NUMERIC(10, 2),
    closing_balance NUMERIC(10, 2),
    carry_forward_sum NUMERIC(10, 2),
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);