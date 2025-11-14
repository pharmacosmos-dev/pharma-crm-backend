DROP FUNCTION IF EXISTS calculate_import_totals();
DROP FUNCTION IF EXISTS trigger_imports_status_change() CASCADE;
DROP FUNCTION IF EXISTS trigger_import_details_change();
DROP TRIGGER IF EXISTS trg_update_imports_from_details ON imports;