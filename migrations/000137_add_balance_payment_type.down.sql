DELETE FROM cashbox_payment_types
WHERE payment_type_id IN (
    SELECT id FROM payment_types WHERE type = 'balance'
);

DELETE FROM payment_types
WHERE type = 'balance';