ALTER TABLE "transfers" 
ALTER COLUMN "public_id" 
DROP DEFAULT;

DROP SEQUENCE IF EXISTS transfers_public_id_seq;