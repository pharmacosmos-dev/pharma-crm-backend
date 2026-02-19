CREATE TABLE IF NOT EXISTS employee_targets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_target_id UUID NOT NULL REFERENCES store_targets(id),
    employee_id UUID NOT NULL REFERENCES employees(id),
    store_id UUID NOT NULL REFERENCES stores(id),
    company_id UUID NOT NULL,
    amount NUMERIC(20, 2) NOT NULL DEFAULT 0,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(employee_id, year, month)
);