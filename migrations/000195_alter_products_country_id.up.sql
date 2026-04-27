-- Migrate existing country string values into countries table
INSERT INTO countries (name)
SELECT DISTINCT country
FROM products
WHERE country IS NOT NULL AND country != ''
ON CONFLICT (name) DO NOTHING;

-- Add country_id FK column
ALTER TABLE products ADD COLUMN IF NOT EXISTS country_id UUID REFERENCES countries(id);

-- Populate country_id from existing country values
UPDATE products p
SET country_id = c.id
FROM countries c
WHERE p.country = c.name;

-- Drop old country string column
ALTER TABLE products DROP COLUMN IF EXISTS country;
