ALTER TABLE "payment_types"
    ADD COLUMN "front_name" varchar(225);

UPDATE "payment_types"
SET "front_name" =
    CASE
        WHEN name = 'Uzcard' THEN 'uzcard'
        WHEN name = 'Humo' THEN 'humo'
        WHEN name = 'Click' THEN 'click'
        WHEN name = 'Naqd' THEN 'cash'
        WHEN name = 'Uzum' THEN 'uzum'
        WHEN name = 'Payme' THEN 'payme'
        WHEN name = 'Balans' THEN 'loyalty_card'
        WHEN name = 'Alif' THEN 'alif'
        ELSE ''
    END;