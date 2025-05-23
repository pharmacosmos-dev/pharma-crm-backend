ALTER TABLE import_details ADD COLUMN IF NOT EXISTS "barcode" VARCHAR(20);
ALTER TABLE import_details ADD COLUMN IF NOT EXISTS "store_product_id" UUID REFERENCES store_products(id); 
ALTER TABLE import_details ADD COLUMN IF NOT EXISTS imported_at TIMESTAMP;