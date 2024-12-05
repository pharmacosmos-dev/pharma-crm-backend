CREATE TABLE IF NOT EXISTS "drafts" (
    "id" UUID NOT NULL PRIMARY KEY,
    "product_id" UUID REFERENCES products(id),
    "cash_box_id" UUID REFERENCES cash_boxes(id),
    "customer_id" UUID REFERENCES customers(id),
    "draft_number" VARCHAR(10),
    "sale_id" UUID REFERENCES sales(id),
    "quantity" INT,
    "unit_price" NUMERIC(10, 2),
    "total_price" NUMERIC(10, 2),
    "description" TEXT,
    "draft_time" TIMESTAMP,
    "created_by" UUID,
    "updated_by" UUID,
    "deleted_by" UUID,
    "is_active" BOOLEAN NOT NULL DEFAULT true,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);