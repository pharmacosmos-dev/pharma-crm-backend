ALTER TABLE
    "imports"
        ADD COLUMN IF NOT EXISTS "received_count" NUMERIC(20, 2) DEFAULT 0.0,
        ADD COLUMN IF NOT EXISTS "received_sum" NUMERIC(20, 2) DEFAULT 0.0,
        ADD COLUMN IF NOT EXISTS "scanned_count" NUMERIC(20, 2) DEFAULT 0.0,
        ADD COLUMN IF NOT EXISTS "scanned_sum" NUMERIC(20, 2) DEFAULT 0.0;


UPDATE imports i
SET 
    received_count = t.received_count,
    received_sum   = t.received_sum,
    scanned_count  = t.scanned_count,
    scanned_sum    = t.scanned_sum
FROM (
    SELECT 
        import_id,
        COALESCE(SUM(received_count), 0) AS received_count,
        COALESCE(SUM(received_count * retail_price_vat), 0) AS received_sum,
        COALESCE(SUM(scanned_count), 0) AS scanned_count,
        COALESCE(SUM(scanned_count * retail_price_vat), 0) AS scanned_sum
    FROM import_details
    GROUP BY import_id
) AS t
WHERE i.id = t.import_id
  AND i.entry_type = 1;




UPDATE imports i
SET 
    received_count = t.received_count,
    received_sum   = t.received_sum,
    scanned_count  = t.scanned_count,
    scanned_sum    = t.scanned_sum
FROM (
    SELECT 
        d.import_id,
        COALESCE(SUM(d.received_count / p.unit_per_pack), 0) AS received_count,
        COALESCE(SUM((d.received_count / p.unit_per_pack) * d.retail_price_vat), 0) AS received_sum,
        COALESCE(SUM(d.scanned_count / p.unit_per_pack), 0) AS scanned_count,
        COALESCE(SUM((d.scanned_count / p.unit_per_pack) * d.retail_price_vat), 0) AS scanned_sum
    FROM import_details d
    JOIN products p ON p.id = d.product_id
    GROUP BY d.import_id
) AS t
WHERE i.id = t.import_id
  AND i.entry_type = 2;

        