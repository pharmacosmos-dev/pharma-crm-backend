CREATE TABLE IF NOT EXISTS cash_box_histories (
    id UUID NOT NULL PRIMARY KEY,
    cash_box_id UUID REFERENCES cash_boxes(id),
    cash_amount NUMERIC(10, 2),
    cashless_amount NUMERIC(10, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);