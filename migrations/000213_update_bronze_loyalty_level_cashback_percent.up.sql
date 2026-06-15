UPDATE loyalty_card_levels SET cashback_percent = 3 WHERE position = 1;
ALTER TABLE customers ALTER COLUMN loyalty_card_percent SET DEFAULT 3;
