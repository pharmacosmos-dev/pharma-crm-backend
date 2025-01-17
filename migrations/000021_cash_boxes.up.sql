CREATE TABLE IF NOT EXISTS "cash_boxes" (
    "id" UUID NOT NULL PRIMARY KEY,
    "store_id" UUID REFERENCES stores(id),
    "name" VARCHAR(255),
    "is_open" BOOLEAN,
    "is_enable" BOOLEAN,
    "created_by" UUID,
    "updated_by" UUID,
    "deleted_by" UUID,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMPTZ
);