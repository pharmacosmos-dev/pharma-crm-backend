CREATE TABLE IF NOT EXISTS "sale_items" (
    "id" UUID NOT NULL PRIMARY KEY,
    "product_id" UUID REFERENCES products(id),
    "sale_id" UUID REFERENCES sales(id),
    "employee_id" UUID REFERENCES employees(id),
    "quantity" INT NOT NULL,
    "drug_count" INT,
    "unit_price" NUMERIC(10, 2), -- Base price from products
    "discount_type" VARCHAR(10) CHECK (discount_type IN ('percent', 'cash')),
    "discount_value" NUMERIC(10, 2) DEFAULT 0,
    "discount_amount" NUMERIC(10, 2),
    "total_price" NUMERIC(10, 2),
    "total_discount_price" NUMERIC(10, 2),
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);