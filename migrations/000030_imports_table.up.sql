CREATE TABLE IF NOT EXISTS "imports" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "public_id" INT,
    "store_id" UUID REFERENCES stores("id"),
    "status" VARCHAR(55) NOT NULL DEFAULT 'pending',
    "import_date" TIMESTAMP,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "import_details" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "import_id" UUID REFERENCES imports("id"),
    "product_id" UUID REFERENCES products("id"),
    "received_count" INT DEFAULT 0,
    "accepted_count" INT DEFAULT 0,
    "canceled_count" INT DEFAULT 0,
    "received_amount" DECIMAL(10, 2) DEFAULT 0.00,
    "accepted_amount" DECIMAL(10, 2) DEFAULT 0.00,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);