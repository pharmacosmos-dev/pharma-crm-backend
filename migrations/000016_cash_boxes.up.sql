CREATE TABLE IF NOT EXISTS "cash_boxes" (
    "id" UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "store_id" UUID REFERENCES stores(id),
    "name" VARCHAR(255),
    "is_open" BOOLEAN,
    "is_enable" BOOLEAN,
    "created_by" UUID,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);