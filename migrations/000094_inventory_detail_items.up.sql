CREATE TABLE IF NOT EXISTS inventory_detail_items (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    import_detail_id UUID REFERENCES import_details(id),
    store_product_id UUID REFERENCES store_products(id),
    current_count NUMERIC(10, 4) DEFAULT 0.0000,
    fact_count NUMERIC(10, 4) DEFAULT 0.0000,
    expire_date DATE,
    retail_price NUMERIC(10, 2) DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
