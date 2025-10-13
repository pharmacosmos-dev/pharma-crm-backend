CREATE TABLE IF NOT EXISTS "update_packaging_logs" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "product_id" UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    "employee_id" UUID NOT NULL REFERENCES employees (id) ON DELETE SET NULL,
    "old_unit_per_pack" INTEGER NOT NULL,
    "new_unit_per_pack" INTEGER NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
