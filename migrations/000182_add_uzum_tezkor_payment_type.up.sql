INSERT INTO "payment_types" ("name", "type", "description", "front_name", "is_active", "order_number")
VALUES ('UzumTezkor', 'online_order', 'Uzum Tezkor orqali to''lov', 'uzum_tezkor', true, 9)
ON CONFLICT DO NOTHING;
