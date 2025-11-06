-- Modified function with better logic
CREATE OR REPLACE FUNCTION calculate_import_totals(p_import_id INTEGER)
RETURNS VOID AS $$
DECLARE
    v_entry_type INTEGER;
    v_status VARCHAR;
BEGIN
    -- Import ma'lumotlarini olish
    SELECT entry_type, status 
    INTO v_entry_type, v_status
    FROM imports 
    WHERE id = p_import_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- entry_type = 1 (dona hisobida)
    IF v_entry_type = 1 THEN
        -- NEW status uchun received hisoblash
        UPDATE imports
        SET 
            received_count = COALESCE((
                SELECT SUM(d.received_count)
                FROM import_details d
                WHERE d.import_id = p_import_id
            ), 0),
            received_sum = COALESCE((
                SELECT SUM(d.received_count * d.retail_price_vat)
                FROM import_details d
                WHERE d.import_id = p_import_id
            ), 0),
            updated_at = NOW()
        WHERE id = p_import_id;
        
        -- COMPLETED status uchun scanned hisoblash
        IF v_status = 'completed' THEN
            UPDATE imports
            SET 
                scanned_count = COALESCE((
                    SELECT SUM(COALESCE(d.scanned_count, 0))
                    FROM import_details d
                    WHERE d.import_id = p_import_id
                ), 0),
                scanned_sum = COALESCE((
                    SELECT SUM(COALESCE(d.scanned_count, 0) * d.retail_price_vat)
                    FROM import_details d
                    WHERE d.import_id = p_import_id
                ), 0),
                updated_at = NOW()
            WHERE id = p_import_id;
        END IF;
    END IF;

    -- entry_type = 2 (quti/pack hisobida)
    IF v_entry_type = 2 THEN
        UPDATE imports
        SET 
            received_count = COALESCE((
                SELECT SUM(d.received_count / NULLIF(p.unit_per_pack, 0))
                FROM import_details d
                JOIN products p ON p.id = d.product_id
                WHERE d.import_id = p_import_id
            ), 0),
            received_sum = COALESCE((
                SELECT SUM((d.received_count / NULLIF(p.unit_per_pack, 0)) * d.retail_price_vat)
                FROM import_details d
                JOIN products p ON p.id = d.product_id
                WHERE d.import_id = p_import_id
            ), 0),
            updated_at = NOW()
        WHERE id = p_import_id;
        
        IF v_status = 'completed' THEN
            UPDATE imports
            SET 
                scanned_count = COALESCE((
                    SELECT SUM(COALESCE(d.scanned_count, 0) / NULLIF(p.unit_per_pack, 0))
                    FROM import_details d
                    JOIN products p ON p.id = d.product_id
                    WHERE d.import_id = p_import_id
                ), 0),
                scanned_sum = COALESCE((
                    SELECT SUM((COALESCE(d.scanned_count, 0) / NULLIF(p.unit_per_pack, 0)) * d.retail_price_vat)
                    FROM import_details d
                    JOIN products p ON p.id = d.product_id
                    WHERE d.import_id = p_import_id
                ), 0),
                updated_at = NOW()
            WHERE id = p_import_id;
        END IF;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- 3. imports jadvalidagi status o'zgarganda trigger
CREATE OR REPLACE FUNCTION trigger_imports_status_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Faqat status o'zgarganda ishlasin
    IF TG_OP = 'UPDATE' AND NEW.status IS DISTINCT FROM OLD.status THEN
        PERFORM calculate_import_totals(NEW.id);
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_imports_status_change
AFTER UPDATE OF status
ON imports
FOR EACH ROW
EXECUTE FUNCTION trigger_imports_status_change();

-- 4. import_details jadvalidagi o'zgarishlar uchun trigger
CREATE OR REPLACE FUNCTION trigger_import_details_change()
RETURNS TRIGGER AS $$
DECLARE
    v_import_id INTEGER;
BEGIN
    -- O'zgargan import_id ni aniqlash
    IF TG_OP = 'DELETE' THEN
        v_import_id := OLD.import_id;
    ELSE
        v_import_id := NEW.import_id;
    END IF;
    
    -- Hisoblashni bajarish
    PERFORM calculate_import_totals(v_import_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_imports_from_details
AFTER INSERT OR UPDATE OR DELETE
ON import_details
FOR EACH ROW
EXECUTE FUNCTION trigger_import_details_change();