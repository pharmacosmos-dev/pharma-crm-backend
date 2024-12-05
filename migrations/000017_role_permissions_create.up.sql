CREATE TABLE IF NOT EXISTS "role_permissions" (
    "id" uuid PRIMARY KEY,
    "role_id" uuid REFERENCES "roles"("id"),
    "permission_id" uuid REFERENCES "permissions"("id"),
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);