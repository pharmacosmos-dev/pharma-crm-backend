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