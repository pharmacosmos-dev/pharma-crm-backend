-- Xodim ma'lumotini oxirgi marta kim o'zgartirgani (status Уволен/Активный uchun)
ALTER TABLE "employees"
    ADD COLUMN IF NOT EXISTS "updated_by" UUID REFERENCES "employees" ("id") ON DELETE SET NULL;
