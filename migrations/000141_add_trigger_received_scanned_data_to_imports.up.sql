CREATE OR REPLACE FUNCTION update_imports_totals()
RETURNS TRIGGER AS $$
BEGIN
    -- Faqat status yoki entry_type o‘zgarganda yoki yangi yozuv qo‘shilganda ishlaydi
    IF (TG_OP = 'INSERT' OR TG_OP = 'UPDATE') THEN

        -- entry_type = 1 bo‘lsa
        IF NEW.entry_type = 1 THEN

            -- status = 'new' holatida received_count va received_sum ni hisoblaymiz
            IF NEW.status = 'new' THEN
                UPDATE imports
                SET received_count = COALESCE((
                        SELECT SUM(d.received_count)
                        FROM import_details d
                        WHERE d.import_id = NEW.id
                    ), 0),
                    received_sum = COALESCE((
                        SELECT SUM(d.received_count * d.retail_price_vat)
                        FROM import_details d
                        WHERE d.import_id = NEW.id
                    ), 0),
                    updated_at = NOW()
                WHERE id = NEW.id;

            -- status = 'completed' holatida scanned_count va scanned_sum ni hisoblaymiz
            ELSIF NEW.status = 'completed' THEN
                UPDATE imports
                SET scanned_count = COALESCE((
                        SELECT SUM(d.scanned_count)
                        FROM import_details d
                        WHERE d.import_id = NEW.id
                    ), 0),
                    scanned_sum = COALESCE((
                        SELECT SUM(d.scanned_count * d.retail_price_vat)
                        FROM import_details d
                        WHERE d.import_id = NEW.id
                    ), 0),
                    updated_at = NOW()
                WHERE id = NEW.id;
            END IF;
        END IF;


        -- entry_type = 2 bo‘lsa
        IF NEW.entry_type = 2 THEN

            -- status = 'new' holatida (unit_per_pack bilan)
            IF NEW.status = 'new' THEN
                UPDATE imports
                SET received_count = COALESCE((
                        SELECT SUM(d.received_count / p.unit_per_pack)
                        FROM import_details d
                        JOIN products p ON p.id = d.product_id
                        WHERE d.import_id = NEW.id
                    ), 0),
                    received_sum = COALESCE((
                        SELECT SUM((d.received_count / p.unit_per_pack) * d.retail_price_vat)
                        FROM import_details d
                        JOIN products p ON p.id = d.product_id
                        WHERE d.import_id = NEW.id
                    ), 0),
                    updated_at = NOW()
                WHERE id = NEW.id;

            -- status = 'completed' holatida (unit_per_pack bilan)
            ELSIF NEW.status = 'completed' THEN
                UPDATE imports
                SET scanned_count = COALESCE((
                        SELECT SUM(d.scanned_count / p.unit_per_pack)
                        FROM import_details d
                        JOIN products p ON p.id = d.product_id
                        WHERE d.import_id = NEW.id
                    ), 0),
                    scanned_sum = COALESCE((
                        SELECT SUM((d.scanned_count / p.unit_per_pack) * d.retail_price_vat)
                        FROM import_details d
                        JOIN products p ON p.id = d.product_id
                        WHERE d.import_id = NEW.id
                    ), 0),
                    updated_at = NOW()
                WHERE id = NEW.id;
            END IF;
        END IF;

    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER trg_update_imports_totals
AFTER INSERT OR UPDATE
ON imports
FOR EACH ROW
EXECUTE FUNCTION update_imports_totals();
