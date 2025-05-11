CREATE TABLE IF NOT EXISTS tax_products(
    id SERIAL PRIMARY KEY,
    "name_uz" VARCHAR(500),
    "name_ru" VARCHAR(500),
    "mxik" VARCHAR(25),
    "unit_code" VARCHAR(15),
    "unit_name" VARCHAR(255),
    "unit_count" INT DEFAULT 1,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);