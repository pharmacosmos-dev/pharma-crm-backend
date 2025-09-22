CREATE TABLE IF NOT EXISTS "product_barcodes" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "product_id" UUID NOT NULL REFERENCES "products" ("id") ON DELETE CASCADE,
    "old_barcode" VARCHAR(50),
    "barcode" VARCHAR(50) NOT NULL,
    "created_by" UUID REFERENCES "employees" ("id") ON DELETE SET NULL,
    "status" VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, completed
    "store_id" UUID REFERENCES "stores" ("id"),
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);