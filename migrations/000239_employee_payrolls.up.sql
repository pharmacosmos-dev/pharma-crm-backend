CREATE TABLE IF NOT EXISTS "employee_payrolls" (
    "id"                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "employee_id"              UUID NOT NULL REFERENCES "employees" ("id"),
    "store_id"                 UUID REFERENCES "stores" ("id"),
    "position_snapshot"        VARCHAR(100),
    "experience_years"         NUMERIC(5,1)  NOT NULL DEFAULT 0,
    "worked_hours"             NUMERIC(10,2) NOT NULL DEFAULT 0,
    "avg_monthly_hours"        NUMERIC(10,2) NOT NULL DEFAULT 0,
    "salary_rate_amount"       NUMERIC(20,2) NOT NULL DEFAULT 0,
    "actual_salary_amount"     NUMERIC(20,2) NOT NULL DEFAULT 0,
    "individual_sales_amount"  NUMERIC(20,2) NOT NULL DEFAULT 0,
    "store_target_id"          UUID REFERENCES "store_targets" ("id"),
    "store_plan_amount"        NUMERIC(20,2) NOT NULL DEFAULT 0,
    "kpi_percent"              NUMERIC(5,2)  NOT NULL DEFAULT 0,
    "kpi_amount"               NUMERIC(20,2) NOT NULL DEFAULT 0,
    "bonus_amount"             NUMERIC(20,2) NOT NULL DEFAULT 0,
    "gross_salary_amount"      NUMERIC(20,2) NOT NULL DEFAULT 0,
    "advance_card_amount"      NUMERIC(20,2) NOT NULL DEFAULT 0,
    "advance_cash_amount"      NUMERIC(20,2) NOT NULL DEFAULT 0,
    "deduction_term_amount"    NUMERIC(20,2) NOT NULL DEFAULT 0,
    "deduction_recount_amount" NUMERIC(20,2) NOT NULL DEFAULT 0,
    "deduction_fine_amount"    NUMERIC(20,2) NOT NULL DEFAULT 0,
    "net_pay_amount"           NUMERIC(20,2) NOT NULL DEFAULT 0,
    "status"                   VARCHAR(20) NOT NULL DEFAULT 'draft',
    "approved_by"              UUID REFERENCES "employees" ("id"),
    "month"                    INTEGER NOT NULL,
    "year"                     INTEGER NOT NULL,
    "created_at"               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at"               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "completed_at"             TIMESTAMP WITH TIME ZONE,
    CONSTRAINT uq_employee_payrolls_employee_year_month UNIQUE ("employee_id", "year", "month")
);

CREATE INDEX IF NOT EXISTS idx_employee_payrolls_store_year_month ON employee_payrolls (store_id, year, month);
CREATE INDEX IF NOT EXISTS idx_employee_payrolls_status ON employee_payrolls (status);
CREATE INDEX IF NOT EXISTS idx_employee_payrolls_store_target_id ON employee_payrolls (store_target_id);
