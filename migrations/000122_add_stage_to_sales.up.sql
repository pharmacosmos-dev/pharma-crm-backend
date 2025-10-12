ALTER TABLE 
    "sales"
        ADD COLUMN IF NOT EXISTS "stage" INT DEFAULT 1 NOT NULL,
        ADD COLUMN IF NOT EXISTS "display_id" BIGINT,
        ADD COLUMN IF NOT EXISTS "otp_code" VARCHAR(255) NULL;


UPDATE "sales"
SET "display_id" = "sale_number"
WHERE "display_id" IS NULL;

ALTER TABLE "sales"
    ALTER COLUMN "display_id" SET NOT NULL;