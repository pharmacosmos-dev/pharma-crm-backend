CREATE TABLE IF NOT EXISTS product_receipts (
    id UUID NOT NULL PRIMARY KEY,
    document_number VARCHAR(50) UNIQUE,
    document_date DATE, 
    total_sum NUMERIC(10, 2),
    total_vat_sum NUMERIC(10, 2),
    store_code INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);