CREATE TABLE IF NOT EXISTS finance_categories(
    "id" SERIAL PRIMARY KEY,
    "parent_id" INT REFERENCES finance_categories(id) ON DELETE CASCADE,
    "name" VARCHAR(55),
    "description" TEXT,
    "account_group" VARCHAR(55),
    "status" VARCHAR(55),
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);