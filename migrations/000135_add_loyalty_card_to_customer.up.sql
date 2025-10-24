-- Create loyalty_card_level table
CREATE TABLE loyalty_card_levels (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "name" VARCHAR(20) UNIQUE NOT NULL,
    "min_spent" NUMERIC(10, 2) DEFAULT 0.00,
    "cashback_percent" INTEGER NOT NULL,
    "position" INTEGER NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);

-- Create loyalty_card_levelup_history table
CREATE TABLE loyalty_card_levelup_history (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "customer_id" UUID REFERENCES "customers"("id") ON DELETE CASCADE,
    "loyalty_card_level_id" UUID REFERENCES "loyalty_card_levels"("id") ON DELETE CASCADE,
    "total_spent" NUMERIC(10, 2) DEFAULT 0.00,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

ALTER TABLE 
    "customers"
        ADD COLUMN "loyalty_card_barcode" VARCHAR(20) UNIQUE,
        ADD COLUMN "loyalty_card_percent" INT DEFAULT 0,
        ADD COLUMN "loyalty_card_level_id" UUID REFERENCES "loyalty_card_levels"("id"),
        ADD COLUMN "loyalty_card_type" VARCHAR,
        ADD COLUMN "loyalty_card_created_by" UUID REFERENCES "employees"("id"),
        ADD COLUMN "telegram_chat_id" INT UNIQUE;