ALTER TABLE online_price_revaluation_details
    ADD COLUMN IF NOT EXISTS "store_id" UUID REFERENCES stores(id) ON DELETE CASCADE;
