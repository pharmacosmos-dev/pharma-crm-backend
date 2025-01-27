CREATE TABLE IF NOT EXISTS auto_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID REFERENCES stores(id),
    product_id UUID REFERENCES products(id),
    suggested_order INT NOT NULL,
    adjusted_order INT DEFAULT NULL, -- Manager-adjusted value
    finalized BOOLEAN DEFAULT FALSE, -- Indicates if order is finalized
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()  
);