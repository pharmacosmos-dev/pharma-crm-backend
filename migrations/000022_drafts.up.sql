CREATE SEQUENCE IF NOT EXISTS "draft_number_seq" START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS "drafts" (
    "id" UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    "draft_number" INTEGER NOT NULL DEFAULT nextval('draft_number_seq'),
    "product_id" UUID REFERENCES products(id) ON DELETE CASCADE,
    "cash_box_id" UUID REFERENCES cash_boxes(id) ON DELETE CASCADE,
    "customer_id" UUID REFERENCES customers(id) ON DELETE CASCADE,
    "sale_id" UUID REFERENCES sales(id),
    "description" TEXT,
    "status" VARCHAR(50) DEFAULT 'pending',
    "created_by" UUID,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "draft_time" TIMESTAMP,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);