CREATE TABLE IF NOT EXISTS online_price_revaluations (
    "id"            SERIAL PRIMARY KEY,
    "store_id"      UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    "platform_type" VARCHAR(50) NOT NULL,             -- 'uzum', 'yandex_eda', ...
    "name"          VARCHAR(255),
    "status"        VARCHAR(25) NOT NULL DEFAULT 'new', -- new | pending | completed | canceled
    "created_by"    UUID REFERENCES employees(id) ON DELETE SET NULL,
    "updated_by"    UUID REFERENCES employees(id) ON DELETE SET NULL,
    "created_at"    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at"    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS online_price_revaluation_details (
    "id"                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "online_price_revaluation_id" INT NOT NULL REFERENCES online_price_revaluations(id) ON DELETE CASCADE,
    "product_id"                  UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    "old_retail_price"            NUMERIC(10, 2) NOT NULL DEFAULT 0,
    "new_retail_price"            NUMERIC(10, 2) NOT NULL DEFAULT 0,
    "old_supply_price"            NUMERIC(10, 2) NOT NULL DEFAULT 0,
    "created_at"                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at"                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (online_price_revaluation_id, product_id)
);
