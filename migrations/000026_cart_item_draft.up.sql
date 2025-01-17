CREATE TABLE IF NOT EXISTS "cart_item_drafts" (
    "id" UUID PRIMARY KEY,
    "cart_item_id" UUID REFERENCES "cart_items"("id"),
    "draft_id" UUID REFERENCES "drafts"("id"),
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);