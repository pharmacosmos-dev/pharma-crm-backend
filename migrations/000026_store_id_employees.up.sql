ALTER TABLE employees 
    ADD COLUMN IF NOT EXISTS store_id uuid REFERENCES stores(id);