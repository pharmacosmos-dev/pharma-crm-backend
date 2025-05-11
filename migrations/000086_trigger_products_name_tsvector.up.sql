CREATE FUNCTION IF NOT EXISTS update_name_tsvector() RETURNS trigger AS $$
BEGIN
  NEW.name_tsvector := to_tsvector('simple', NEW.name);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER IF NOT EXISTS trg_update_name_tsvector
BEFORE INSERT OR UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION update_name_tsvector();