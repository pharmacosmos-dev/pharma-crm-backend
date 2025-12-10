ALTER TABLE 
    "categories"
        ADD COLUMN IF NOT EXISTS "name_uz" VARCHAR(255),
        ADD COLUMN IF NOT EXISTS "name_kr" VARCHAR(255),
        ADD COLUMN IF NOT EXISTS "name_en" VARCHAR(255);