CREATE TABLE IF NOT EXISTS "cashbox_payment_types" (
    "id" UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "cash_box_id" UUID REFERENCES cash_boxes(id),
    "payment_type_id" UUID REFERENCES payment_types(id),
    "is_active" BOOLEAN DEFAULT true,
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);