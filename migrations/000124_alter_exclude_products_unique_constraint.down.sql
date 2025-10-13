DO $$
    BEGIN
        -- yangi constraintni olib tashlaymiz
        IF EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'unique_excluded_product'
              AND conrelid = 'excluded_products'::regclass
        ) THEN
            ALTER TABLE excluded_products
                DROP CONSTRAINT unique_excluded_product;
        END IF;

        -- eski constraintni tiklaymiz (store_id, product_id)
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'excluded_products_store_id_product_id_key'
              AND conrelid = 'excluded_products'::regclass
        ) THEN
            ALTER TABLE excluded_products
                ADD CONSTRAINT excluded_products_store_id_product_id_key UNIQUE (store_id, product_id);
        END IF;
    END
$$;
