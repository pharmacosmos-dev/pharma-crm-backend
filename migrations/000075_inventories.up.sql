CREATE TABLE IF NOT EXISTS inventories(
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(), 
    "public_id" VARCHAR(20), 
    "store_id" UUID NOT NULL REFERENCES stores(id),
    "name" VARCHAR(255),
    "type" VARCHAR(55) DEFAULT 'FULL', -- FULL || PARTIAL || IMPORT
    "status" INT DEFAULT 0, -- 0 -> new, 1 -> pending, 2 -> completed
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inventory_details(
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(), 
    "inventory_id" UUID NOT NULL REFERENCES inventories(id) ON DELETE CASCADE,
    "product_id" UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    "scanned_count" INT DEFAULT 0,
    "accepted_count" INT DEFAULT 0,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);