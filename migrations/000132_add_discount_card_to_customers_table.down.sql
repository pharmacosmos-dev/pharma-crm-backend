ALTER TABLE 
    "customers"
        DROP COLUMN IF EXISTS "discount_card",
        DROP COLUMN IF EXISTS "discount_percent";


ALTER TABLE 
    "sale_customer_discounts"
        ADD CONSTRAINT IF NOT EXISTS "sale_customer_discounts_sale_id_customer_id_discount_card_id_key";

ALTER TABLE 
    "sale_customer_discounts"
        DROP CONSTRAINT IF EXISTS "sale_customer_discounts_sale_id_customer_id_key" UNIQUE (sale_id, customer_id);
