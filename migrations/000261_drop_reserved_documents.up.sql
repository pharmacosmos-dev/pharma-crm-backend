-- Rezerv hujjati reserves / reserve_details (000260) da olib boriladi.
-- Eski 000258_reserved_documents migratsiyasi yaratgan jadvallar endi ishlatilmaydi.
-- DIQQAT: bu jadvallardagi ma'lumot butunlay o'chadi.
DROP TABLE IF EXISTS "reserved_details";
DROP TABLE IF EXISTS "reserved";
