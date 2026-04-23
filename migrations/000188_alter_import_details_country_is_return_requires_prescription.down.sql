ALTER TABLE "import_details"
    DROP COLUMN IF EXISTS "country",
    DROP COLUMN IF EXISTS "is_return",
    DROP COLUMN IF EXISTS "requires_prescription";
