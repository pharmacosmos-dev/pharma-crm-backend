CREATE TABLE IF NOT EXISTS "companies" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "name" VARCHAR(255),
    "logo" TEXT,
    "email" VARCHAR(255),
    "phone" VARCHAR(20),
    "country" VARCHAR(255),
    "city" VARCHAR(255),
    "legal_name" VARCHAR(255),
    "legal_address" VARCHAR(255),
    "postal_code" VARCHAR(10),
    "company_inn" VARCHAR(55),
    "company_stir" VARCHAR(255),
    "company_mfo" VARCHAR(255),
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);