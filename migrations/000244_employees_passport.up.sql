-- Xodimning passport/ID karta raqami. Hozircha IXTIYORIY (NULL bo'lishi mumkin):
-- mavjud xodimlarda bu ma'lumot yo'q va create/update'da ham majburiy emas.
-- Keyinchalik majburiy qilinganda avval mavjud qatorlar to'ldirilib, so'ng
-- alohida migratsiyada NOT NULL qo'yiladi.
ALTER TABLE "employees" ADD COLUMN IF NOT EXISTS "passport" VARCHAR(50);
