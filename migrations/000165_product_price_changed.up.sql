CREATE TABLE IF NOT EXISTS "product_price_changed" (
  "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  "store_code" INT,
  "barcode" VARCHAR(255),
  "markirovka" VARCHAR[],
  "material_code" INT,
  "product_series_number" VARCHAR(255),
  "sum" NUMERIC(18,4) DEFAULT 0,
  "sum_vat" NUMERIC(18,4) DEFAULT 0,
  "supply_price" NUMERIC(18,4) DEFAULT 0,
  "supply_price_vat" NUMERIC(18,4) DEFAULT 0,
  "vat" VARCHAR(50),
  "vat_price" NUMERIC(18,4) DEFAULT 0,
  "vat_sum" NUMERIC(18,4) DEFAULT 0,
  "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
