CREATE TABLE IF NOT EXISTS "transfers" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "public_id" VARCHAR(20),
    "name" VARCHAR(255),
    "comment" VARCHAR(500),
    "from_store_id" UUID REFERENCES stores("id") ON DELETE CASCADE,
    "to_store_id" UUID REFERENCES stores("id") ON DELETE CASCADE,
    "entry_type" INTEGER DEFAULT 1, -- 1 - transfer || 2 - return 
    "status" VARCHAR(55) NOT NULL DEFAULT 'new', -- new || pending || completed || canceled || write-off
    "created_by" UUID REFERENCES employees("id"),
    "updated_by" UUID REFERENCES employees("id"),
    "accepted_by" UUID REFERENCES employees("id"),
    "accepted_at" TIMESTAMP,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS "transfer_details" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "transfer_id" UUID REFERENCES transfers("id") ON DELETE CASCADE,
    "product_id" UUID REFERENCES products("id") ON DELETE CASCADE,
    "store_product_id" UUID,
    "received_count" INT DEFAULT 0,
    "accepted_count" INT DEFAULT 0,
    "scanned_count" INT DEFAULT 0,
    "expire_date" TIMESTAMP,
    "serial_number" VARCHAR(255),
    "supply_price" NUMERIC(20, 2) DEFAULT 0.00,
    "retail_price" NUMERIC(20, 2) DEFAULT 0.00,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);