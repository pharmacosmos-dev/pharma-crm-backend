CREATE SEQUENCE IF NOT EXISTS "imports_public_id_seq" START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS "imports" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "public_id" INTEGER NOT NULL DEFAULT nextval('imports_public_id_seq'),
    "store_id" UUID REFERENCES stores("id") ON DELETE CASCADE,
    "store_code" INTEGER,
    "document_number" VARCHAR(50),
    "created_by" UUID REFERENCES employees("id"),
    "accepted_by" UUID REFERENCES employees("id"),
    "status" VARCHAR(55) NOT NULL DEFAULT 'new', -- new || pending || completed || canceled || write-off
    "import_date" TIMESTAMP,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "document_year" INT GENERATED ALWAYS AS (EXTRACT(YEAR FROM "import_date")) STORED
);

-- Add the unique constraint if it does not already exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_constraint 
        WHERE conname = 'unique_document_number_year'
    ) THEN
        ALTER TABLE "imports" 
        ADD CONSTRAINT "unique_document_number_year" 
        UNIQUE ("document_number", "document_year");
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS "import_details" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "import_id" UUID REFERENCES imports("id") ON DELETE CASCADE,
    "product_id" UUID REFERENCES products("id") ON DELETE CASCADE,
    "received_count" INT DEFAULT 0,
    "accepted_count" INT DEFAULT 0,
    "canceled_count" INT DEFAULT 0,
    "expire_date" TIMESTAMP,
    "supply_price" NUMERIC(20, 2) DEFAULT 0.00,
    "vat" INT DEFAULT 0,
    "vat_sum" NUMERIC(10, 2) DEFAULT 0.00,
    "retail_price" NUMERIC(20, 2) DEFAULT 0.00,
    "accepted_retail_price" NUMERIC(20, 2) DEFAULT 0.00,
    "series_number" VARCHAR(255),
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);