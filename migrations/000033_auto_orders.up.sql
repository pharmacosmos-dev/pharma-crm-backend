CREATE SEQUENCE IF NOT EXISTS "auto_orders_public_id_seq" START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS auto_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID REFERENCES stores(id),
    public_id INTEGER NOT NULL DEFAULT nextval('auto_orders_public_id_seq'),
    status VARCHAR(20) DEFAULT 'new', -- pending, completed, canceled
    auto_order_date TIMESTAMP,
    completed_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS auto_order_details (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auto_order_id UUID REFERENCES auto_orders(id),
    product_id UUID REFERENCES products(id),
    kvant INT DEFAULT 0,
    current_stock INT DEFAULT 0,
    min_stock INT DEFAULT 0,
    max_stock INT DEFAULT 0,
    month_sale_stock INT DEFAULT 0,
    day_sale_stock INT DEFAULT 0,
    order_growth FLOAT DEFAULT 0,
    order_lead_time FLOAT DEFAULT 0,
    suggested_order_quantity INT DEFAULT 0,
    adjusted_order_quantity INT DEFAULT 0,
    response_order_quantity INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
)