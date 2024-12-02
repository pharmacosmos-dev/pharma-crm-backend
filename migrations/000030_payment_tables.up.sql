CREATE TABLE IF NOT EXISTS payment_types (
    id UUID NOT NULL PRIMARY KEY,
    name VARCHAR(255),
    type VARCHAR(10),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payment_services (
    id UUID NOT NULL PRIMARY KEY,
    store_id UUID REFERENCES stores(id),
    name VARCHAR(255),
    merchant_id INT,
    service_id INT,
    secret_key VARCHAR(255),
    is_active BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sale_payments (
    id UUID NOT NULL PRIMARY KEY,
    sale_id UUID REFERENCES sales(id),
    payment_service_id UUID REFERENCES payment_services(id),
    payment_type_id UUID REFERENCES payment_types(id),
    amount NUMERIC(10, 2),
    paid_at TIMESTAMP,
    status VARCHAR(20),
    transaction_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID NOT NULL PRIMARY KEY,
    sale_payment_id UUID REFERENCES sale_payments(id),
    payment_service_id UUID REFERENCES payment_services(id),
    transaction_id UUID,
    status VARCHAR(20),
    response_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);