ALTER TABLE
    "customers"
        ADD COLUMN "spending_from_balance" NUMERIC(10, 2) DEFAULT 0.00,
        ADD COLUMN "loyalty_card_created_at" TIMESTAMP WITH TIME ZONE;