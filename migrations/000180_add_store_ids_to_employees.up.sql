ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS store_ids text[] NOT NULL DEFAULT '{}';
