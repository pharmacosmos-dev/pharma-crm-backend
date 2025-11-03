CREATE OR REPLACE FUNCTION update_imports_totals()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT' OR TG_OP = 'UPDATE') THEN

        IF NEW.entry_type = 1 THEN
            IF NEW.status = 'new' THEN
                SELECT
                    COALESCE(SUM(d.received_count), 0),
                    COALESCE(SUM(d.received_count * d.retail_price_vat), 0)
                INTO NEW.received_count, NEW.received_sum
                FROM import_details d
                WHERE d.import_id = NEW.id;

            ELSIF NEW.status = 'completed' THEN
                SELECT
                    COALESCE(SUM(d.scanned_count), 0),
                    COALESCE(SUM(d.scanned_count * d.retail_price_vat), 0)
                INTO NEW.scanned_count, NEW.scanned_sum
                FROM import_details d
                WHERE d.import_id = NEW.id;
            END IF;
        END IF;

        IF NEW.entry_type = 2 THEN
            IF NEW.status = 'new' THEN
                SELECT
                    COALESCE(SUM(d.received_count / p.unit_per_pack), 0),
                    COALESCE(SUM((d.received_count / p.unit_per_pack) * d.retail_price_vat), 0)
                INTO NEW.received_count, NEW.received_sum
                FROM import_details d
                JOIN products p ON p.id = d.product_id
                WHERE d.import_id = NEW.id;

            ELSIF NEW.status = 'completed' THEN
                SELECT
                    COALESCE(SUM(d.scanned_count / p.unit_per_pack), 0),
                    COALESCE(SUM((d.scanned_count / p.unit_per_pack) * d.retail_price_vat), 0)
                INTO NEW.scanned_count, NEW.scanned_sum
                FROM import_details d
                JOIN products p ON p.id = d.product_id
                WHERE d.import_id = NEW.id;
            END IF;
        END IF;

        NEW.updated_at := NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_imports_totals
BEFORE INSERT OR UPDATE
ON imports
FOR EACH ROW
EXECUTE FUNCTION update_imports_totals();