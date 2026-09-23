-- Rezerv hujjati va uning qatorlari — imports / import_details sxemasi bilan bir xil:
-- "reserves" hujjat (do'kon, hujjat raqami, status, jami miqdorlar),
-- "reserve_details" o'sha hujjatdagi mahsulotlar.
-- Manba ro'yxat 1C dan keladi (reserved_products), hujjat esa do'konda yig'iladi.

-- Hujjat raqami avtomatik: RZ-1000, RZ-1001, ... (imports'dagi public_id sequence kabi).
CREATE SEQUENCE IF NOT EXISTS "reserves_dok_number_seq" START WITH 1000 INCREMENT BY 1 MINVALUE 1000;

CREATE TABLE IF NOT EXISTS "reserves" (
    "id"                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "dok_number"          VARCHAR(50) NOT NULL UNIQUE DEFAULT ('RZ-' || nextval('reserves_dok_number_seq')),
    "store_id"            UUID NOT NULL REFERENCES stores("id") ON DELETE CASCADE,
    -- new — yangi yaratilgan, checking — yig'ilyapti/tekshirilyapti, done — yakunlangan
    "status"              VARCHAR(20) NOT NULL DEFAULT 'new' CHECK ("status" IN ('new', 'checking', 'done')),
    -- Qatorlardan hisoblanadi: har bir yozish/o'chirishdan keyin qayta yoziladi.
    "total_quantity"      NUMERIC(20, 2) NOT NULL DEFAULT 0,
    "total_product_count" INTEGER NOT NULL DEFAULT 0,
    "comment"             TEXT,
    "created_by"          UUID REFERENCES employees("id"),
    "updated_by"          UUID REFERENCES employees("id"),
    "completed_at"        TIMESTAMP WITH TIME ZONE,
    "created_at"          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at"          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Ro'yxat do'kon va status bo'yicha filtrlanadi, sana bo'yicha saralanadi.
CREATE INDEX IF NOT EXISTS idx_reserves_store_status ON reserves (store_id, status);
CREATE INDEX IF NOT EXISTS idx_reserves_created_at ON reserves (created_at DESC);

CREATE TABLE IF NOT EXISTS "reserve_details" (
    "id"                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    "reserve_id"          UUID NOT NULL REFERENCES reserves("id") ON DELETE CASCADE,
    "product_id"          UUID NOT NULL REFERENCES products("id") ON DELETE CASCADE,
    -- 1C ro'yxatidagi qaysi qatordan olingani; ro'yxat yangilanib qator yo'qolsa NULL bo'ladi.
    "reserved_product_id" UUID REFERENCES reserved_products("id") ON DELETE SET NULL,
    -- Snapshot: mahsulot keyin o'zgarsa ham hujjatda o'sha paytdagi kod va nom qoladi.
    "material_code"       VARCHAR(100) NOT NULL DEFAULT '',
    "product_name"        VARCHAR(500) NOT NULL DEFAULT '',
    -- quantity — rejalashtirilgan rezerv, checked_quantity — checking bosqichida topilgani.
    "quantity"            NUMERIC(20, 2) NOT NULL DEFAULT 0,
    "checked_quantity"    NUMERIC(20, 2) NOT NULL DEFAULT 0,
    "created_by"          UUID REFERENCES employees("id"),
    "updated_by"          UUID REFERENCES employees("id"),
    "created_at"          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    "updated_at"          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    -- Bitta hujjatda bitta mahsulot bir marta: qayta qo'shilsa miqdori yangilanadi (upsert).
    CONSTRAINT uq_reserve_details_reserve_product UNIQUE ("reserve_id", "product_id")
);

CREATE INDEX IF NOT EXISTS idx_reserve_details_reserve_id ON reserve_details (reserve_id);
CREATE INDEX IF NOT EXISTS idx_reserve_details_product_id ON reserve_details (product_id);
