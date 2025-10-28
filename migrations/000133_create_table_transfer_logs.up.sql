CREATE TABLE IF NOT EXISTS "transfer_logs" (
    "id"                    BIGSERIAL PRIMARY KEY,
    "transfer_id"           UUID    NOT NULL REFERENCES "transfers"("id") ON DELETE CASCADE,
    "transfer_detail_id"    UUID    NOT NULL REFERENCES "transfer_details"("id") ON DELETE CASCADE,
    "user_id"               UUID    NOT NULL REFERENCES "employees"("id") ON DELETE SET NULL,
    "product_id"            UUID    NOT NULL REFERENCES "products"("id") ON DELETE SET NULL,
    "quantity"              INT     NULL,
    "stage"                 SMALLINT DEFAULT 0 NOT NULL, -- 0 - created, 1 - sent, 2 - received, 3 - checking, 4 - completed
    "transfer_type"         SMALLINT NOT NULL DEFAULT 0, -- 0 - transfer, 1 - return
    "created_at"            TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "updated_at"            TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);