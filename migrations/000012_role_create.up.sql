CREATE TABLE IF NOT EXISTS "roles" (
  "id" uuid PRIMARY KEY,
  "name" varchar,
  "description" text,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "permissions" (
    "id" uuid PRIMARY KEY,
    "entity_name" VARCHAR(50) NOT NULL, 
    "action" VARCHAR(20) NOT NULL,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()       
);

