CREATE TABLE IF NOT EXISTS "cart_items" (
    "id" UUID NOT NULL PRIMARY KEY,
    "product_id" UUID REFERENCES products(id),
    "sale_id" UUID REFERENCES sales(id),
    "employee_id" UUID REFERENCES employees(id),
    "quantity" INT NOT NULL,
    "unit_price" NUMERIC(10, 2), -- Base price from products
    "discount_type" VARCHAR(10) CHECK (discount_type IN ('percent', 'cash')),
    "discount_value" NUMERIC(10, 2) DEFAULT 0,
    "discount_amount" NUMERIC(10, 2) GENERATED ALWAYS AS (
        CASE 
            WHEN discount_type = 'percent' THEN unit_price * discount_value / 100
            WHEN discount_type = 'cash' THEN discount_value
            ELSE 0
        END
    ) STORED, -- Calculated discount
    "total_price" NUMERIC(10, 2) GENERATED ALWAYS AS (
        CASE 
            WHEN discount_type = 'percent' THEN (unit_price - (unit_price * discount_value / 100)) * quantity
            WHEN discount_type = 'cash' THEN (unit_price - discount_value) * quantity
            ELSE unit_price * quantity
        END
    ) STORED, -- Price after discount
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);