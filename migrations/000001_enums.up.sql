CREATE TYPE  "product_type_enum" AS ENUM (
  'product',
  'service',
  'set'
);

CREATE TYPE  "product_variability" AS ENUM (
  'basic',
  'variative'
);

CREATE TYPE  "product_status" AS ENUM (
  'active',
  'inactive',
  'low_stock',
  'zero_stock',
  'expired',
  'deleted' 
);

CREATE TYPE  "order_status" AS ENUM (
  'pending',
  'completed',
  'canceled'
);

CREATE EXTENSION IF NOT EXISTS "pgcrypto";