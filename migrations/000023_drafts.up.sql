CREATE SEQUENCE IF NOT EXISTS "draft_number_seq" START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS "drafts" (
    "id" UUID NOT NULL PRIMARY KEY,
    "draft_number" INTEGER NOT NULL DEFAULT nextval('draft_number_seq'),
    "product_id" UUID REFERENCES products(id),
    "cash_box_id" UUID REFERENCES cash_boxes(id),
    "customer_id" UUID REFERENCES customers(id),
    "sale_id" UUID REFERENCES sales(id),
    "description" TEXT,
    "draft_time" TIMESTAMP,
    "status" VARCHAR(50) DEFAULT 'pending',
    "created_by" UUID,
    "updated_by" UUID,
    "deleted_by" UUID,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);