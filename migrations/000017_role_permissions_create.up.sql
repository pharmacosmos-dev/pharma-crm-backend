CREATE TABLE IF NOT EXISTS "role_permissions" (
    "id" uuid PRIMARY KEY,
    "role_id" uuid REFERENCES "roles"("id"),
    "permission_id" uuid REFERENCES "permissions"("id"),
    "is_active" boolean NOT NULL DEFAULT true,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);