-- An earlier revision of this migration created `reserveds` and then failed.
-- Rename that partial table before retrying migration 258.
DO $$
BEGIN
    IF to_regclass('public.reserveds') IS NOT NULL
       AND to_regclass('public.reserved') IS NULL THEN
        ALTER TABLE "reserveds" RENAME TO "reserved";
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS "reserved" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "store_id" UUID NOT NULL REFERENCES stores("id") ON DELETE CASCADE,
    "document_number" VARCHAR(50) NOT NULL,
    "created_by" UUID REFERENCES employees("id") ON DELETE SET NULL,
    "total_quantity" NUMERIC(14, 4) NOT NULL DEFAULT 0,
    "total_product_count" INTEGER NOT NULL DEFAULT 0,
    "status" VARCHAR(20) NOT NULL DEFAULT 'new'
        CHECK ("status" IN ('new', 'checking', 'done')),
    "comment" VARCHAR(500),
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reserved_store_document_number UNIQUE ("store_id", "document_number"),
    CONSTRAINT chk_reserved_total_quantity_non_negative CHECK ("total_quantity" >= 0),
    CONSTRAINT chk_reserved_total_product_count_non_negative CHECK ("total_product_count" >= 0)
);

CREATE INDEX IF NOT EXISTS idx_reserved_store_status_created_at
    ON "reserved" ("store_id", "status", "created_at" DESC);

CREATE TABLE IF NOT EXISTS "reserved_details" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "reserved_id" UUID NOT NULL REFERENCES "reserved"("id") ON DELETE CASCADE,
    "product_id" UUID NOT NULL REFERENCES products("id") ON DELETE CASCADE,
    "quantity" NUMERIC(14, 4) NOT NULL DEFAULT 0,
    "created_by" UUID REFERENCES employees("id") ON DELETE SET NULL,
    "updated_by" UUID REFERENCES employees("id") ON DELETE SET NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reserved_details_reserved_product UNIQUE ("reserved_id", "product_id"),
    CONSTRAINT chk_reserved_details_quantity_non_negative CHECK ("quantity" >= 0)
);

CREATE INDEX IF NOT EXISTS idx_reserved_details_product_id
    ON "reserved_details" ("product_id");
