CREATE TABLE IF NOT EXISTS "sale_payment_summary" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "cash_box_operation_id" UUID NOT NULL,
    "payment_type_id" UUID NOT NULL,
    "total_amount" DECIMAL(10, 2) DEFAULT 0,
    "total_expense_amount" DECIMAL(10, 2) DEFAULT 0,
    "total_net_amount" DECIMAL(10, 2) DEFAULT 0,
    "total_difference" DECIMAL(10, 2) DEFAULT 0,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE ("cash_box_operation_id", "payment_type_id")
);
