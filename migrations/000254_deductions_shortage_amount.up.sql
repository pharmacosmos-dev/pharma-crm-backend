-- Kutilgan taqsimot summasi: pereuchyotdan chiqqan kamomad.
--
-- Bu bo'lmaganda detallar yig'indisi hech narsaga solishtirilmasdi va
-- 15 mln kamomadga 20 mln taqsimlab yuborish mumkin edi.
--
-- 0 = tekshirilmaydi (masalan shtraf: oldindan ma'lum summa yo'q).
ALTER TABLE "deductions"
    ADD COLUMN IF NOT EXISTS "shortage_amount" NUMERIC(20,2) NOT NULL DEFAULT 0;
