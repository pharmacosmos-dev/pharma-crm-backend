-- Add company_id to products
ALTER TABLE IF EXISTS products
    ADD COLUMN IF NOT EXISTS company_id uuid REFERENCES companies ON DELETE CASCADE;
-- Add company_id to store_products
ALTER TABLE IF EXISTS store_products
    ADD COLUMN IF NOT EXISTS company_id uuid REFERENCES companies ON DELETE CASCADE;
-- Add company_id to employees
ALTER TABLE IF EXISTS employees
    ADD COLUMN IF NOT EXISTS company_id uuid REFERENCES companies ON DELETE CASCADE;
-- Add company_id to excluded_products
ALTER TABLE IF EXISTS excluded_products
    ADD COLUMN IF NOT EXISTS company_id uuid REFERENCES companies ON DELETE CASCADE;
-- Add company_id to product_bonuses
ALTER TABLE IF EXISTS product_bonuses
    ADD COLUMN IF NOT EXISTS company_id uuid REFERENCES companies ON DELETE CASCADE;
