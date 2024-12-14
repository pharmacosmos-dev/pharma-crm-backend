CREATE TABLE IF NOT EXISTS "roles" (
  "id" uuid PRIMARY KEY,
  "public_id"   INT,
  "name" varchar,
  "description" text,
  "created_by" uuid,
  "updated_by" uuid,
  "deleted_by" uuid,
  "status" INT NOT NULL DEFAULT 1, -- 1 - active, 0 - inactive, 2 - deleted
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "permissions" (
    "id" uuid PRIMARY KEY,
    "route" VARCHAR(255) NOT NULL,
    "entity_name" VARCHAR(50) NOT NULL, 
    "type" VARCHAR(20) NOT NULL,
    "action" VARCHAR(20) NOT NULL,
    "parent_id" uuid REFERENCES "permissions"("id"),
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()       
);

