CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transfer_logs_transfer_detail_id
ON transfer_logs (transfer_detail_id);
