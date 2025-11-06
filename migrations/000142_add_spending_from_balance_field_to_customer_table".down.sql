ALTER TABLE
    "customers"
        DROP COLUMN IF EXISTS "spending_from_balance",
        DROP COLUMN IF EXISTS "loyalty_card_created_at";