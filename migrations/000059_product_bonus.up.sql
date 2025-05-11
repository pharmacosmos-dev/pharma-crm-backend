CREATE TABLE IF NOT EXISTS product_bonuses (
    "id" SERIAL PRIMARY KEY,
    "product_id" UUID REFERENCES products(id) ON DELETE CASCADE,
    "store_id" UUID REFERENCES stores(id) ON DELETE CASCADE,
    "bonus_amount" NUMERIC(10, 2) DEFAULT 0.00,
    "status" INT DEFAULT 1, -- 1 = active, 0 = inactive
    "start_date" DATE,
    "end_date" DATE,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);