CREATE SEQUENCE IF NOT EXISTS "auto_orders_public_id_seq" START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS auto_orders (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "store_id" UUID REFERENCES stores(id) ON DELETE CASCADE,
    "public_id" INTEGER NOT NULL DEFAULT nextval('auto_orders_public_id_seq'),
    "created_by" UUID REFERENCES employees(id) ON DELETE CASCADE,
    "updated_by" UUID REFERENCES employees(id) ON DELETE CASCADE,
    "status" VARCHAR(20) DEFAULT 'new', -- pending, completed, canceled
    "auto_order_date" TIMESTAMP,
    "completed_date" TIMESTAMP,
    "created_at" TIMESTAMP DEFAULT NOW(),
    "updated_at" TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS auto_order_details (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "auto_order_id" UUID REFERENCES auto_orders(id) ON DELETE CASCADE,
    "product_id" UUID REFERENCES products(id) ON DELETE CASCADE,
    "kvant" INT DEFAULT 1,
    "current_stock" NUMERIC(10, 4) DEFAULT 0.0000,
    "min_stock" NUMERIC(10, 4) DEFAULT 0.0000,
    "max_stock" NUMERIC(10, 4) DEFAULT 0.0000,
    "sale_count" NUMERIC(10, 4) DEFAULT 0.0000,
    "daily_sale_count" NUMERIC(10, 4) DEFAULT 0.0000,
    "import_day" INT DEFAULT 2,
    "sale_period" INT DEFAULT 3,
    "stock_on_delivery_date" NUMERIC(10, 4) DEFAULT 0.0000,
    "reserve_quantity" NUMERIC(10, 4) DEFAULT 0.0000,
    "future_stock" NUMERIC(10, 4) DEFAULT 0.0000,
    "future_stock_with_reserve" NUMERIC(10, 4) DEFAULT 0.0000,
    "order_count" NUMERIC(10, 4) DEFAULT 0.0000,
    "created_at" TIMESTAMP DEFAULT NOW(),
    "updated_at" TIMESTAMP DEFAULT NOW()
);