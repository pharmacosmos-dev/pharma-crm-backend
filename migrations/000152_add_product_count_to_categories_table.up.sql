ALTER TABLE 
    "categories"
        ADD COLUMN IF NOT EXISTS
            "product_count" INT DEFAULT 0;


UPDATE categories c
SET product_count = COALESCE(p.cnt, 0)
FROM (
    SELECT category_id, COUNT(*) AS cnt
    FROM products
    GROUP BY category_id
) p
WHERE c.id = p.category_id;
