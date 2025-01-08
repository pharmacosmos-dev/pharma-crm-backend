CREATE TABLE IF NOT EXISTS employee_roles (
    "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    "employee_id" uuid REFERENCES employees(id),
    "role_id" uuid REFERENCES roles(id),
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);