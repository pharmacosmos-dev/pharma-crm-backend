CREATE TYPE product_movement_type AS ENUM (
  'IMPORT',
  'SALE',
  'RETURN_SALE',
  'RETURN_SUPPLIER',
  'TRANSFER_OUT',
  'TRANSFER_IN',
  'INVENTORY',
  'REPRICING'
);


CREATE TABLE IF NOT EXISTS product_movements(
  "id"             BIGSERIAL  PRIMARY KEY,
  "product_id"     UUID       NOT NULL     REFERENCES products(id),
  "store_id"       UUID       DEFAULT NULL REFERENCES stores(id),
  "to_store_id"    UUID       DEFAULT NULL REFERENCES stores(id),
  "movement_type"  product_movement_type NOT NULL,
  "movement_id"    UUID           NOT NULL,
  "display_id"     BIGINT         NOT NULL DEFAULT 0,
  "prev_quantity"  INTEGER        DEFAULT 0,
  "quantity"       INTEGER        DEFAULT 0,
  "after_quantity" INTEGER        DEFAULT 0,
  "price"          NUMERIC(14, 2) DEFAULT 0.00,
  "total_price"    NUMERIC(14, 2) DEFAULT 0.00,
  "status"         SMALLINT       DEFAULT 0, -- 0 -> NEW, 1 -> PENDING, -1 -> CANCELLED, 2 -> SENDING, 3 -> COMPLETED
  "movement_date"  TIMESTAMP      NOT NULL DEFAULT NOW(),
  "created_at"     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);