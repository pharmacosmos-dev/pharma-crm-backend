ALTER TABLE
    "cart_items"
        ADD COLUMN IF NOT EXISTS "skip_auto_order" BOOLEAN DEFAULT FALSE;