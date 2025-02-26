CREATE TABLE IF NOT EXISTS product_markings(
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "import_detail_id" UUID REFERENCES import_details(id) ON DELETE CASCADE,
    "product_id" UUID REFERENCES products(id) ON DELETE CASCADE,
    "marking" VARCHAR(500),
    "status" INT DEFAULT 0,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);