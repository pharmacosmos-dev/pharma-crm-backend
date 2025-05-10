CREATE TABLE IF NOT EXISTS "products" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "brand_id" UUID REFERENCES "brands"("id"),
  "unit_type_id" UUID REFERENCES "unit_types"("id"),
  "shelf_id" UUID REFERENCES "shelves"("id"),
  "producer_id" UUID REFERENCES "producers"("id"),
  "material_code" INT UNIQUE,
  "name" VARCHAR(255),
  "barcode" VARCHAR(255),
  "photos" VARCHAR[],
  "unit_per_pack" INTEGER DEFAULT 1,
  "description" TEXT,
  "status" VARCHAR(55) DEFAULT 'active' CHECK ("status" IN ('active', 'deleted', 'inactive')),
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
  "deleted_at" TIMESTAMP
);