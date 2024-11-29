CREATE TABLE IF NOT EXISTS cash_boxes (
    id UUID NOT NULL PRIMARY KEY,
    store_id UUID REFERENCES stores(id),
    name VARCHAR(255),
    is_open BOOLEAN,
    is_enable BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);