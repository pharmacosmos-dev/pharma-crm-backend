ALTER TABLE role_permissions
DROP CONSTRAINT role_permissions_permission_id_fkey;

ALTER TABLE role_permissions
ADD CONSTRAINT role_permissions_permission_id_fkey
FOREIGN KEY (permission_id)
REFERENCES permissions(id)
ON DELETE CASCADE;