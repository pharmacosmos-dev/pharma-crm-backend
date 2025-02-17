CREATE TABLE IF NOT EXISTS "suppliers" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "name" VARCHAR(255),
  "phone" VARCHAR(20),
  "desc" TEXT,
  "company_legal_name" VARCHAR(255),
  "legal_address" VARCHAR(255),
  "country" VARCHAR(255),
  "zip_code" VARCHAR(50),
  "bank_account" VARCHAR(100),
  "bank_name" VARCHAR(255),
  "bank_tin" VARCHAR(255),
  "bank_ibt" VARCHAR(255),
  "file_url" VARCHAR(1000),
  "is_active" BOOLEAN NOT NULL DEFAULT true,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "deleted_at" TIMESTAMP
);