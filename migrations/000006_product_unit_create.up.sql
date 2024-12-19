CREATE TABLE IF NOT EXISTS unit_types(
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "unit_name" VARCHAR(50) NOT NULL,
  "codename" VARCHAR(50),
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "product_units" (
  "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  "product_id" UUID NOT NULL REFERENCES "products"(id),
  "unit_type_id" UUID NOT NULL REFERENCES "unit_types"(id),
  "unit_name" VARCHAR(20),
  "box_grain_count" INT DEFAULT 0,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);