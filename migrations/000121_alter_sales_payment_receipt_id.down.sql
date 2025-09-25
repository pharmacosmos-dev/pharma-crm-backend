ALTER TABLE 
    "sales"
        DROP COLUMN IF EXISTS "is_sent_to_tax",
        DROP COLUMN IF EXISTS "payment_receipt_id";