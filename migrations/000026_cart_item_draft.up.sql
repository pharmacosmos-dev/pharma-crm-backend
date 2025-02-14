CREATE TABLE IF NOT EXISTS "cart_item_drafts" (
    "id" UUID PRIMARY KEY,
    "cart_item_id" UUID REFERENCES "cart_items"("id") ON DELETE CASCADE,
    "draft_id" UUID REFERENCES "drafts"("id") ON DELETE CASCADE,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);