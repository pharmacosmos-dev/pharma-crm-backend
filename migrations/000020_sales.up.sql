CREATE SEQUENCE IF NOT EXISTS "sale_number_seq" START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS "sales" (
    "id" UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "sale_number" INTEGER NOT NULL DEFAULT nextval('sale_number_seq'),
    "employee_id" UUID REFERENCES employees(id),
    "cash_box_operation_id" UUID REFERENCES cashbox_operations(id),
    "customer_id" UUID REFERENCES customers(id),
    "discount_type" VARCHAR(10),
    "total_discount" NUMERIC(10, 2),
    "total_amount" NUMERIC(10, 2),
    "created_by" UUID,
    "status" VARCHAR(20) DEFAULT 'pending',
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "completed_at" TIMESTAMP,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);
