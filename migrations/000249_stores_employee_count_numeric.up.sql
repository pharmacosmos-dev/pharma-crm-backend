-- Do'konning rejadagi xodimlar soni yarim stavkani ham ko'rsatishi kerak (2.5),
-- shuning uchun INT emas, NUMERIC.
ALTER TABLE "stores"
    ALTER COLUMN "employee_count" TYPE NUMERIC(6,2) USING COALESCE("employee_count", 0)::NUMERIC(6,2);

ALTER TABLE "stores"
    ALTER COLUMN "employee_count" SET DEFAULT 0;
