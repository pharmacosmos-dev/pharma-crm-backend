CREATE TABLE IF NOT EXISTS "rejected_products" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "product_id" UUID REFERENCES "products" ("id") ON DELETE CASCADE,
    "product_name" VARCHAR(255),
    "store_id" UUID NOT NULL REFERENCES "stores" ("id") ON DELETE CASCADE,
    "reason" TEXT,
    "rejected_times" NUMERIC(10, 2),
    "created_by" UUID REFERENCES "employees" ("id") ON DELETE SET NULL,
    "updated_by" UUID REFERENCES "employees" ("id") ON DELETE SET NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);