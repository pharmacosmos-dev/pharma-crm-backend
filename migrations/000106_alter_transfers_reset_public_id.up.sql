CREATE SEQUENCE IF NOT EXISTS transfers_public_id_seq 
START WITH 1000 
INCREMENT BY 1 
MINVALUE 1000;

ALTER TABLE "transfers" 
ALTER COLUMN "public_id" 
SET DEFAULT nextval('transfers_public_id_seq');
