INSERT INTO payment_types (name, type, description, is_active, order_number)
VALUES ('Balans', 'loyalty_cd', 'Loyalty karta ortali to`lov', true, 8);

INSERT INTO cashbox_payment_types (cash_box_id, payment_type_id, is_active)
SELECT c.id, p.id, true
FROM cash_boxes c
         JOIN payment_types p ON p.type = 'loyalty_cd'
WHERE NOT EXISTS (
    SELECT 1
    FROM cashbox_payment_types cp
    WHERE cp.cash_box_id = c.id
      AND cp.payment_type_id = p.id
);

ALTER TABLE IF EXISTS sales
    ADD COLUMN IF NOT EXISTS loyalty_card NUMERIC(10, 2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cash_back NUMERIC(10, 2) DEFAULT 0;

