CREATE SEQUENCE IF NOT EXISTS roles_public_id_seq START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS "roles" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "public_id" INTEGER NOT NULL DEFAULT nextval('roles_public_id_seq'),
  "name" VARCHAR(255),
  "description" TEXT,
  "status" INT NOT NULL DEFAULT 1, -- 1 - active, 0 - inactive, 2 - deleted
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "deleted_at" TIMESTAMP
);

CREATE TABLE IF NOT EXISTS "permissions" (
    "id" uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    "route" VARCHAR(255) NOT NULL,
    "name" VARCHAR(255) NOT NULL, 
    "type" VARCHAR(20) NOT NULL,
    "key" VARCHAR(50),
    "method" VARCHAR[],
    "parent_id" uuid REFERENCES "permissions"("id") ON DELETE CASCADE,
    "description" TEXT,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP       
);

