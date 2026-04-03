ALTER TABLE "cash_boxes" DROP COLUMN IF EXISTS "terminal_id";
ALTER TABLE "stores" ADD COLUMN "terminal_id" VARCHAR(255)[];
