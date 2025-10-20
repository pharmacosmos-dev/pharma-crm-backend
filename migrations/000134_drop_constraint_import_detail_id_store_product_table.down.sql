ALTER TABLE store_products
ADD CONSTRAINT store_products_import_detail_id_fkey
FOREIGN KEY (import_detail_id)
REFERENCES import_details(id)
ON DELETE SET NULL;