INSERT INTO loyalty_card_levels (
    "name",
    "min_spent",
    "cashback_percent",
    "position",
    "created_at"
) VALUES 
    ('Bronze', 0.00, 1, 1, NOW()),
    ('Silver', 1000000.00, 3, 2, NOW()),
    ('Gold', 5000000.00, 5, 3, NOW());