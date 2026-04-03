ALTER TABLE "stores" DROP COLUMN IF EXISTS "terminal_id";
ALTER TABLE "cash_boxes" ADD COLUMN "terminal_id" VARCHAR(255);
