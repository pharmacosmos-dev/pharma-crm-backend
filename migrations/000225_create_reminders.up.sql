CREATE TABLE IF NOT EXISTS "reminders" (
    "id"          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "text"        TEXT NOT NULL,
    "from_date"   TIMESTAMP WITH TIME ZONE NOT NULL,
    "to_date"     TIMESTAMP WITH TIME ZONE NOT NULL,
    "store_ids"   TEXT[] NOT NULL DEFAULT '{}',
    "created_by"  UUID REFERENCES "employees" ("id"),
    "created_at"  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at"  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reminders_to_date ON reminders (to_date);
CREATE INDEX IF NOT EXISTS idx_reminders_created_by ON reminders (created_by);
CREATE INDEX IF NOT EXISTS idx_reminders_store_ids ON reminders USING GIN (store_ids);
