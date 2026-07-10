CREATE INDEX IF NOT EXISTS idx_imports_entry_status_store_created
ON imports (entry_type, status, store_id, created_at);

CREATE INDEX IF NOT EXISTS idx_import_details_import_product
ON import_details (import_id, product_id);
