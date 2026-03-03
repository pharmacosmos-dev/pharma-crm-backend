ALTER TABLE 
    "sales" 
        ADD COLUMN IF NOT EXISTS "vendor_order_id" VARCHAR(55) DEFAULT NULL;