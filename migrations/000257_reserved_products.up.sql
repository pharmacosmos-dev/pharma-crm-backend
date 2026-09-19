CREATE TABLE IF NOT EXISTS "reserved_products" (
    "id"            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "sort_index"    INTEGER NOT NULL,
    "name"          VARCHAR(500) NOT NULL DEFAULT '',
    "material_code" VARCHAR(100) NOT NULL,
    "is_active"     BOOLEAN NOT NULL DEFAULT TRUE,
    "created_at"    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at"    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reserved_products_material_code UNIQUE ("material_code")
);


CREATE INDEX IF NOT EXISTS idx_reserved_products_sort_index ON reserved_products (sort_index);
CREATE INDEX IF NOT EXISTS idx_reserved_products_active_sort ON reserved_products (sort_index) WHERE is_active;
