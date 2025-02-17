CREATE TABLE IF NOT EXISTS "employee_roles" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "employee_id" UUID REFERENCES employees(id) ON DELETE CASCADE,
    "role_id" UUID REFERENCES roles(id) ON DELETE CASCADE,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);