CREATE TABLE IF NOT EXISTS "imports" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "public_id" INT,
    "store_id" UUID REFERENCES stores("id"),
    "store_code" INT REFERENCES stores("store_code"),
    "document_number" VARCHAR(50),
    "created_by" UUID REFERENCES employees("id"),
    "accepted_by" UUID REFERENCES employees("id"),
    "status" VARCHAR(55) NOT NULL DEFAULT 'new', -- new || pending || completed || canceled || 
    "import_date" TIMESTAMP,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "document_year" INT GENERATED ALWAYS AS (EXTRACT(YEAR FROM import_date)) STORED
);

ALTER TABLE "imports" 
ADD CONSTRAINT unique_document_number_year UNIQUE ("document_number", "document_year");

CREATE TABLE IF NOT EXISTS "import_details" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "import_id" UUID REFERENCES imports("id"),
    "product_id" UUID REFERENCES products("id"),
    "product_material_code" INT REFERENCES products("material_code"),
    "received_count" INT DEFAULT 0,
    "accepted_count" INT DEFAULT 0,
    "canceled_count" INT DEFAULT 0,
    "received_amount" DECIMAL(10, 2) DEFAULT 0.00,
    "accepted_amount" DECIMAL(10, 2) DEFAULT 0.00,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);