ALTER TABLE 
    "stores"
        ADD COLUMN IF NOT EXISTS "fixed_stage" INTEGER DEFAULT 0;