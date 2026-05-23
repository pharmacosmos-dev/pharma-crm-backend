ALTER TABLE transfers ADD COLUMN IF NOT EXISTS "comment_by" UUID REFERENCES employees("id");
