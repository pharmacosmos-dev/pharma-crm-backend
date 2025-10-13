ALTER TABLE 
    "cart_items" 
        ADD COLUMN IF NOT EXISTS "is_marking" BOOLEAN DEFAULT false;