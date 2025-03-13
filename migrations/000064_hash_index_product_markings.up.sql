CREATE INDEX IF NOT EXISTS product_markings_marking_hash_idx  ON product_markings USING HASH (marking);
CREATE INDEX IF NOT EXISTS product_markings_product_id_idx ON product_markings (product_id);

