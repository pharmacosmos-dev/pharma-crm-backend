ALTER TABLE
    "customers"
        ADD COLUMN IF NOT EXISTS "discount_card" VARCHAR(20) NULL,
        ADD COLUMN IF NOT EXISTS "discount_percent" INT DEFAULT 0;


ALTER TABLE "sale_customer_discounts"
DROP CONSTRAINT "unique_sale_customer_discount";

ALTER TABLE "sale_customer_discounts"
ADD CONSTRAINT "sale_customer_discounts_sale_id_customer_id_key" UNIQUE ("sale_id", "customer_id");
