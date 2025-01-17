CREATE TABLE IF NOT EXISTS "cart_items" (
    "id" UUID NOT NULL PRIMARY KEY,
    "product_id" UUID REFERENCES products(id),
    "employee_id" UUID REFERENCES employees(id),
    "sale_id" UUID REFERENCES sales(id),
    "store_product_id" UUID REFERENCES store_products(id),
    "quantity" INT DEFAULT 0,
    "unit_quantity" INT DEFAULT 0,
    "unit_price" NUMERIC(10, 2), -- Base price from products
    "discount_type" VARCHAR(10) CHECK ("discount_type" IN ('percent', 'cash')),
    "discount_value" NUMERIC(10, 2) DEFAULT 0,
    "discount_amount" NUMERIC(10, 2),
    "total_price" NUMERIC(10, 2),
    "total_discount_price" NUMERIC(10, 2),
    "status" VARCHAR(20) CHECK ("status" IN ('pending', 'active', 'deleted', 'sold', 'drafted')),
    "is_drafted" BOOLEAN NOT NULL DEFAULT false,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


