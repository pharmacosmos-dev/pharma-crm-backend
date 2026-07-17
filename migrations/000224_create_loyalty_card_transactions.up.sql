CREATE TABLE IF NOT EXISTS "loyalty_card_transactions" (
    "id"                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "sale_id"            UUID NOT NULL REFERENCES "sales" ("id"),
    "customer_id"        UUID NOT NULL REFERENCES "customers" ("id"),
    "type"               VARCHAR(3) NOT NULL CHECK ("type" IN ('in', 'out')),
    "percent"            INTEGER NOT NULL DEFAULT 0,
    "total_sale_amount"  NUMERIC(18, 2) NOT NULL DEFAULT 0.00,
    "old_balance_amount" NUMERIC(18, 2) NOT NULL DEFAULT 0.00,
    "bonus_in_amount"    NUMERIC(18, 2) NOT NULL DEFAULT 0.00,
    "bonus_out_amount"   NUMERIC(18, 2) NOT NULL DEFAULT 0.00,
    "new_balance_amount" NUMERIC(18, 2) NOT NULL DEFAULT 0.00,
    "created_at"         TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_loyalty_card_transactions_customer_id ON loyalty_card_transactions (customer_id);
CREATE INDEX IF NOT EXISTS idx_loyalty_card_transactions_sale_id ON loyalty_card_transactions (sale_id);
CREATE INDEX IF NOT EXISTS idx_loyalty_card_transactions_created_at ON loyalty_card_transactions (created_at);
CREATE INDEX IF NOT EXISTS idx_loyalty_card_transactions_type ON loyalty_card_transactions ("type");
