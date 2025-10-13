DO $$
    BEGIN
        -- old unique constraintni olib tashlaymiz (faqat mavjud bo‘lsa)
        IF EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'excluded_products_store_id_product_id_key'
              AND conrelid = 'excluded_products'::regclass
        ) THEN
            ALTER TABLE excluded_products
                DROP CONSTRAINT excluded_products_store_id_product_id_key;
        END IF;

        -- yangi unique constraint qo‘shamiz (store_id, product_id, company_id)
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'unique_excluded_product'
              AND conrelid = 'excluded_products'::regclass
        ) THEN
            ALTER TABLE excluded_products
                ADD CONSTRAINT unique_excluded_product UNIQUE (store_id, product_id, company_id);
        END IF;
    END
$$;
