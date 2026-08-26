ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "calculated_at";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "elapsed_work_days";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "month_work_days";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "plan_achievement_percent";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "expected_plan_amount";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "employee_plan_amount";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "role_names";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "role";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "store_name";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "company_id";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "full_name";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "last_name";
ALTER TABLE "employee_payrolls" DROP COLUMN IF EXISTS "first_name";

DROP TABLE IF EXISTS "holidays";
