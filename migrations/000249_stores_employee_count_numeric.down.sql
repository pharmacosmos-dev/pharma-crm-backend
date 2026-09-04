-- Kasr qismi yo'qoladi: 2.5 -> 3 (ROUND).
ALTER TABLE "stores"
    ALTER COLUMN "employee_count" TYPE INT USING ROUND(COALESCE("employee_count", 0))::INT;

ALTER TABLE "stores"
    ALTER COLUMN "employee_count" SET DEFAULT 0;
