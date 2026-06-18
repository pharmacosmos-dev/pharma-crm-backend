CREATE INDEX IF NOT EXISTS idx_transfer_details_transfer_id
ON transfer_details (transfer_id);

CREATE INDEX IF NOT EXISTS idx_transfer_details_store_product_id
ON transfer_details (store_product_id);
