CREATE TABLE IF NOT EXISTS discount_cards(
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "customer_id" UUID REFERENCES customers(id),
    "barcode" VARCHAR(13) NOT NULL UNIQUE,
    "percent" INT DEFAULT 0,
    "created_by" UUID REFERENCES employees(id),
    "updated_by" UUID REFERENCES employees(id),
    "deleted_by" UUID REFERENCES employees(id),
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sale_customer_discounts(
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "sale_id" UUID REFERENCES sales(id),
    "customer_id" UUID REFERENCES customers(id),
    "discount_card_id" UUID REFERENCES discount_cards(id),
    "discount_amount" NUMERIC(10, 2) DEFAULT 0.00,
    "discount_percent" INT,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);