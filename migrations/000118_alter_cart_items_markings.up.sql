ALTER TABLE IF EXISTS cart_items
    ADD COLUMN IF NOT EXISTS markings text[] DEFAULT '{}';
