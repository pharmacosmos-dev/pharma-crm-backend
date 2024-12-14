CREATE TABLE IF NOT EXISTS product_receipts (
    id UUID NOT NULL PRIMARY KEY,
    document_number VARCHAR(50) NOT NULL,
    document_date DATE NOT NULL, 
    total_sum NUMERIC(10, 2),
    total_vat_sum NUMERIC(10, 2),
    store_code INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    document_year INT GENERATED ALWAYS AS (EXTRACT(YEAR FROM document_date)) STORED, -- Extract year from document_date
    UNIQUE (document_number, document_year) -- Enforce unique constraint based on document_number and document_year
);
