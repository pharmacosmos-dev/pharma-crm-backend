-- MNN (xalqaro patentlanmagan nom) kodi
ALTER TABLE "products"
    ADD COLUMN IF NOT EXISTS "mnn_code" VARCHAR(255);
