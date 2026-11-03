-- uzum tezkor: kuryer kelish vaqti (courierArrivementDate)
ALTER TABLE sales ADD COLUMN IF NOT EXISTS "courier_arrival_at" TIMESTAMP WITH TIME ZONE;

-- buyurtma yaratilganda store_products'dan olib qo'yilgan miqdor (unit).
-- Yakunlashda faqat farq olinadi, bekor qilinganda shu miqdor qaytariladi.
ALTER TABLE cart_items ADD COLUMN IF NOT EXISTS "reserved_unit_quantity" INT NOT NULL DEFAULT 0;
