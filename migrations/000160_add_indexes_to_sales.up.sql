CREATE INDEX idx_sales_on_completed_at_stage_store_id ON sales(completed_at, stage, store_id);

CREATE INDEX idx_sales_on_store_id_stage ON sales(store_id, stage);