CREATE TABLE IF NOT EXISTS "product_photo_alerts" (
    "id" UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "product_id" UUID NOT NULL REFERENCES "products" ("id") ON DELETE CASCADE,
    "category" SMALLINT NOT NULL CHECK (category IN (1,2,3)), -- 1: ml yoki doza notog'ri; 2: ishlab chiqaruvchi notog'ri; 3: butunlay rasm xato
    "reason" TEXT, -- qisqacha sabab/izoh
    "created_by" UUID REFERENCES "employees" ("id") ON DELETE SET NULL,
    "status" VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, completed (yoki reviewed/resolved)
    "resolved_by" UUID REFERENCES "employees" ("id") ON DELETE SET NULL,
    "resolved_at" TIMESTAMP WITH TIME ZONE,
    "created_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at" TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);