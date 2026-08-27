DROP INDEX IF EXISTS idx_sales_created_at_store;

ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "store_sales_amount";
