ALTER TABLE
    "customers"
        DROP COLUMN IF EXISTS "loyalty_card_barcode",
        DROP COLUMN IF EXISTS "loyalty_card_percent",
        DROP COLUMN IF EXISTS "loyalty_card_level_id",
        DROP COLUMN IF EXISTS "loyalty_card_type",
        DROP COLUMN IF EXISTS "telegram_chat_id";

DROP TABLE IF EXISTS "loyalty_card_levelup_history";
DROP TABLE IF EXISTS "loyalty_card_levels";