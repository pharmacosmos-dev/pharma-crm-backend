-- markings @> ARRAY[...] bo'yicha qidiruv uchun (sotilgan markirovkani tekshirish)
CREATE INDEX IF NOT EXISTS cart_items_markings_gin_idx ON cart_items USING GIN (markings);
