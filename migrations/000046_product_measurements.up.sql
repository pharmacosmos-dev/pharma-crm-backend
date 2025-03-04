CREATE TABLE IF NOT EXISTS product_measurements(
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "mxik_code" VARCHAR(255) UNIQUE,
    "mxik_name_uz" VARCHAR(1000),
    "mxik_name_ru" VARCHAR(1000),
    "unit_name" VARCHAR(255),
    "unit_code" VARCHAR(55),
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);