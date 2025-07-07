CREATE TABLE excluded_products (
                                   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                   store_id UUID REFERENCES stores(id) ON DELETE CASCADE,
                                   product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
                                   created_by UUID REFERENCES employees(id),
                                   updated_by UUID REFERENCES employees(id),
                                   created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
                                   updated_at TIMESTAMP WITHOUT TIME ZONE,
                                   UNIQUE (store_id, product_id)
);

-- Prevent global duplicates (when store_id IS NULL)
CREATE UNIQUE INDEX uniq_excluded_global_product
    ON excluded_products (product_id)
    WHERE store_id IS NULL;
