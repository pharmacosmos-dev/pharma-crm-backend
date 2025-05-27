CREATE TABLE IF NOT EXISTS price_revalutions(
    "id" SERIAL PRIMARY KEY,
    "store_id" UUID REFERENCES stores(id),
    "name" VARCHAR(255),
    "status" VARCHAR(25) DEFAULT 'new', -- new || pending || completed || canceled
    "type" VARCHAR(55) DEFAULT "retail_price", -- supply_price || retail_price || expire_date ...
    "created_by" UUID REFERENCES employees(id),
    "updated_by" UUID REFERENCES employees(id),
    "created_at" TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITHOUT TIME ZONE
);

CREATE TABLE IF NOT EXISTS price_revalution_details(
    "id" SERIAL PRIMARY KEY,
    "price_revalution_id" INT REFERENCES price_revalutions(id),
    "store_product_id" UUID REFERENCES store_products(id),
    "product_id" UUID REFERENCES products(id),
    "old_supply_price" NUMERIC(10, 2) DEFAULT 0.00,
    "new_supply_price" NUMERIC(10, 2) DEFAULT 0.00,
    "old_retail_price" NUMERIC(10, 2) DEFAULT 0.00,
    "new_retail_price" NUMERIC(10, 2) DEFAULT 0.00,
    "old_expire_date" DATE,
    "new_expire_date" DATE,
    "serial_number" VARCHAR(255),
    "created_at" TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW() 
);