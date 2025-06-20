CREATE OR REPLACE FUNCTION update_discount_amount()
RETURNS TRIGGER AS $$
BEGIN
    -- calculate discount amount based on the discount value
    -- if discount_value is not null and greater than 0, calculate discount_amount
    -- otherwise set discount_amount to 0.00
    IF NEW.discount_value IS NOT NULL AND NEW.discount_value > 0 THEN
        NEW.discount_amount := ROUND((NEW.total_price * NEW.discount_value) / 100, 2);
    ELSE
        NEW.discount_amount := 0.00;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER  trg_update_discount_amount
BEFORE INSERT OR UPDATE ON cart_items
FOR EACH ROW
EXECUTE FUNCTION update_discount_amount();