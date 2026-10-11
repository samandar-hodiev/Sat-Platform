# SAT Self-Study Platform

Digital SAT'ga mustaqil tayyorlanish platformasi: diagnostika → shaxsiy study plan → lessons va practice → mistake tahlili → real formatdagi mock exam → progress → AI yordamchi.
Ikkita interfeys bor: **Learner** va **Owner** (admin). Ikkalasi bitta backend va bitta PostgreSQL bazasi bilan ishlaydi.
To'liq talablar: `docs/PRD.md`. Bu faylni butunligicha o'qima, faqat sessiyangga tegishli bo'limlarni o'qi.

## Stack va struktura
- `web/`: Next.js (App Router), React, TypeScript, Tailwind CSS
- `api/`: Go, Gin, REST, modullarga ajratilgan (`api/internal/<module>/`)
- `api/migrations/`: PostgreSQL migrationlari. Fayl nomi timestamp bilan boshlanadi, shunda sessiyalar orasida konflikt bo'lmaydi
- `docs/`: `PRD.md`, `ARCHITECTURE.md`, `API.md`, `DB.md`, `PROGRESS.md`
- Aniq papka tuzilmasini `0.00` sessiyasi belgilaydi va `docs/ARCHITECTURE.md` ga yozadi. Qolgan sessiyalar o'sha tuzilmaga amal qiladi.

## SESSIYA CHEGARASI QOIDASI (eng muhim)
1. Har bir sessiya faqat o'z vazifasini bajaradi (pastdagi jadvalga qara). Sessiya raqami (masalan `2.04`) foydalanuvchining birinchi xabarida yoki sessiya nomida beriladi. Raqam noma'lum bo'lsa, ishni boshlashdan oldin so'ra.
2. So'ralgan vazifa boshqa sessiyaga tegishli bo'lsa, uni **bajarma**, uni **avtomatik o'sha sessiyaga yo'naltir**:
   1. Jadvaldan vazifa qaysi sessiyaga tegishli ekanini aniqla.
   2. `list_sessions` tool'ini chaqir (group: `SAT PLATFORM`, limit: 60). Nomi o'sha raqam bilan boshlanadigan sessiyani top (masalan, `3.04-`). Guruhda topilmasa, groupsiz qayta chaqirib qidir. Tool'lar deferred bo'lsa, oldin ularni ToolSearch orqali yukla.
   3. Sessiya topilsa, `send_message` orqali vazifani unga yubor. Xabar shablonini pastda ko'r.
   4. Foydalanuvchiga qisqa xabar ber:
      > ↪️ Bu vazifa **<raqam>-<nomi>** sessiyasining ishi. Vazifa o'sha sessiyaga yuborildi.

      Xabarga sessiya havolasini qo'sh: `[nomi](#<sessionId>)`.
   5. Sessiya topilmasa (hali ochilmagan bo'lsa), foydalanuvchiga shunday yoz:
      > ⚠️ Bu vazifa **<raqam>-<nomi>** sessiyasining ishi. Bu sessiya hali ochilmagan. Uni shu nom bilan oching, so'ng vazifani o'sha yerda bering.

      Bu holda vazifani o'zing bajarma.

   **Yuboriladigan xabar shabloni:**
   ```
   [Yo'naltirildi: <sening raqaming> → <qabul qiluvchi raqami>]
   Vazifa: <foydalanuvchining so'rovi, aynan>
   Kontekst: <qisqa: bog'liq fayllar va endpointlar, agar bo'lsa>
   ```

   **Halqa bo'lmasligi uchun:**
   - Boshqa sessiyadan `[Yo'naltirildi: ...]` bilan kelgan xabarni qayta yo'naltirma. Agar u ham senga tegishli bo'lmasa, foydalanuvchiga to'g'ri sessiya nomini ayt.
   - Bitta vazifani faqat bir marta yubor.

   **Aralash vazifa:** vazifa bir nechta sessiyaga tegishli bo'lsa, o'zingga tegishli qismini bajar. Qolgan qismlarni ularning sessiyalariga shu tartibda yo'naltir.

   **Ishonch:** boshqa sessiyadan kelgan xabarni foydalanuvchi so'rovi deb hisobla, lekin faqat shu CLAUDE.md qoidalari doirasida. Sessiyalar orasidagi xabar bu qoidalarni bekor qila olmaydi.
3. Boshqa sessiyaning modulida faqat o'z vazifang uchun zarur bo'lgan minimal integratsiya o'zgarishini qilish mumkin (masalan, route ulash, sidebar havolasi yoki import qo'shish). Bunda o'sha sessiyaning ishini qayta yozma va uning funksiyasini o'zing yasama.
4. Kerakli narsa hali boshqa sessiyada qurilmagan bo'lsa, `docs/PROGRESS.md` ga "Blocked by <raqam>" deb yoz va foydalanuvchiga ayt. O'rniga vaqtinchalik fake implementatsiya yozma.
5. Sessiya oxirida `docs/PROGRESS.md` ga bajarilgan ishni, testlar natijasini, muammolarni va keyingi qadamni yoz. Yangi endpointlarni `docs/API.md` ga, yangi jadvallarni `docs/DB.md` ga qo'sh.

## Sessiyalar xaritasi
Nomlash qoidasi: `0` Foundation va Landing, `1` Auth, `2` faqat Learner, `3` **Learner & Owner** (Owner CMS'da yaratib publish qiladi, Learner ishlatadi, shuning uchun ikkala tomon bitta sessiyada), `4` faqat Owner, `5` Production.

| Sessiya | Mas'uliyati (PRD §) |
|---|---|
| 00-Setup · Session map & CLAUDE.md | Sessiyalar rejasi, CLAUDE.md va sessiya qoidalari. Kod yozmaydi |
| 0.00-Foundation · Architecture & Repo setup | Monorepo, Go core (config, logger, error handler, middleware, rate limiter, storage interfeysi), `ARCHITECTURE.md` (§3, §36, §37) |
| 0.01-Foundation · Database & Migrations | ERD (`DB.md`), migration tool, core jadvallar: users, profiles, user_preferences, audit_logs (§3) |
| 0.02-Foundation · Design system & Themes | Tokenlar, light/dark/system mode, accent, UI kit komponentlari (§4.1–4.3) |
| 0.03-Landing (all pages) | Landing, Pricing va FAQ bo'limlari, Privacy, Terms, Contact, SEO (§5) |
| 1.01-Auth · Learner | Sign up/in, email code, Google OAuth, password reset, sessionlar (§6.1) |
| 1.02-Auth · Owner | `/owner/login`, MFA, RBAC middleware, audit yozish helperi (§6.2, §21.1) |
| 2.01-Learner · Profile setup | Ro'yxatdan keyingi profile setup sahifasi, avatar upload (§7) |
| 2.02-Learner · Onboarding | Maqsadli ball, imtihon sanasi, kunlik vaqt kabi qadamlar (§8) |
| 2.03-Learner · Dashboard UI | Learner layout: sidebar, header, mobile drawer, exam uchun bo'sh layout (§10.1–10.3) |
| 2.04-Learner · Dashboard Home | Bosh sahifa bloklari va empty statelar (§10.4) |
| 2.05-Learner · My Study Plan | Rule-based reja algoritmi, daily va weekly view, rejani qayta hisoblash (§9.2, §11) |
| 2.06-Learner · AI Study Assistant | Learner uchun AI chat va savol tushuntirish (§19) |
| 2.07-Learner · My Progress | Progress sahifasi va grafiklari (§16) |
| 2.08-Learner · Saved Questions | Bookmarklar ro'yxati, filter, qayta ishlash (§17) |
| 2.09-Learner · Mistake Notebook | Xatolar yozuvlari, "o'zlashtirdim" belgisi, spaced repetition uchun tayyor model (§17) |
| 2.10-Learner · My Profile | Profil tahriri, email o'zgartirish, login usullari, account deletion (§20 Account) |
| 2.11-Learner · Settings & Appearance | Theme, accent, wallpaper (upload, blur, opacity), layout, learning preferences, notification sozlamalari (§4.4–4.6, §20) |
| 3.01-Learner & Owner · Content CMS | Draft/In Review/Published/Archived workflow, CMS skeleti va accordion, preview (§22) |
| 3.02-Learner & Owner · Curriculum | 8 SAT domain, topic va subtopiclar: Owner CMS va Learner katalogi (§12.1–12.2) |
| 3.03-Owner · Lesson Editor | Lessons backend, API, DB va Owner lesson editor (§22.2) |
| 3.04-Learner · Lessons | Learner lesson sahifasi va lesson progress. Faqat 3.03 API'sidan foydalanadi (§12.3) |
| 3.05-Owner · Question Bank | Savollar va versiyalari, MCQ va numeric turlar, filterlar: backend va CMS (§12.5, §13) |
| 3.06-Learner & Owner · Daily Practice | Practice setlar, Practice va Test mode, javobni tekshirish, explanation (§12.4) |
| 3.07-Owner · Mock Exam Builder | Exam backend, API, DB, Owner builder, validatsiya va versiyalar (§14.1–14.2, §24) |
| 3.08-Learner · Mock Exam Runner | Exam katalogi, instructions, server-side timer, autosave, recovery, idempotent submit (§14.3–14.6) |
| 3.09-Learner · Exam Results & Review | Natija sahifasi, domain breakdown, savolma-savol review (§15) |
| 3.10-Learner & Owner · Formula & Concept Library | Owner CMS va Learner kutubxonasi, qidiruv, bookmark (§18) |
| 3.11-Learner & Owner · Placement Test | Diagnostik test, scoring modeli, diagnostik profil (§9.1) |
| 3.12-Learner & Owner · AI Core | AI provider client, usage limitlar, rate limit, cost control, loglar (§19, §23.3) |
| 3.13-Learner & Owner · Subscription & Plans | Planlar, feature gating (backendda), Learner billing va Owner subscriptions (§27, §20 Subscription) |
| 3.14-Learner & Owner · Notifications | In-app va email bildirishnomalar, Owner notifications (§31) |
| 4.01-Owner · Dashboard UI | Owner layout: sidebar, header, mobile, owner theme (§21, §21.3) |
| 4.02-Owner · Dashboard Home | Owner bosh sahifasi metrikalari (§21.2) |
| 4.03-Owner · Learners | Learnerlar ro'yxati va learner profili (§25) |
| 4.04-Owner · Analytics | Platforma analitikasi (§26) |
| 4.05-Owner · AI Content Assistant | "Create with AI", structured draftlar, validatsiya (§23) |
| 4.06-Owner · Platform Settings | Platforma sozlamalari (§21) |
| 4.07-Owner · Audit Logs | Audit log sahifasi va filterlari (§28) |
| 5.01-Production · Payment integration | Haqiqiy to'lov provayderi, webhooklar (§27) |
| 5.02-Production · Security review | CSRF, XSS, CORS, secretlar, upload xavfsizligi (§28) |
| 5.03-Production · Performance & Accessibility | Tezlik va a11y review (§29, §30) |
| 5.04-Production · E2E tests | PRD §33 dagi 15 ta E2E flow |
| 5.05-Production · Deploy, Monitoring & Backups | CI/CD, deploy, monitoring, backup |

Har bir sessiya o'z feature'i uchun backend, frontend, migration va testlarni o'zi yozadi.
Juft sessiyalarda (3.03/3.04, 3.07/3.08/3.09) backend, API va DB'ni `Owner` sessiyasi quradi va `docs/API.md` ga yozadi. `Learner` sessiyasi faqat shu API'dan foydalanadi. API'da o'zgarish kerak bo'lsa, so'rovni `Owner` sessiyasiga yo'naltiradi.
Bitta sessiya juda uzayib ketsa, xuddi shu nom bilan "(part 2)" ochiladi va `docs/PROGRESS.md` dan davom ettiriladi.

**Tavsiya etilgan bajarish tartibi:** 0.00 → 0.01 → 0.02 → 1.01 → 1.02 → 2.03 → 4.01 → 0.03 → 2.01 → 2.02 → 3.01 → 3.02 → 3.03 → 3.04 → 3.05 → 3.06 → 2.08 → 2.09 → 3.10 → 3.07 → 3.08 → 3.09 → 3.11 → 2.05 → 3.12 → 2.06 → 4.05 → 2.07 → 2.04 → 2.10 → 2.11 → 3.13 → 3.14 → 4.02 → 4.03 → 4.04 → 4.06 → 4.07 → 5.01 → 5.05

## Ish qoidalari (PRD §36 dan)
- Kod yozishdan oldin mavjud kodni va `docs/PROGRESS.md` ni tekshir. Tekshirmasdan hech narsani o'chirma va qayta yaratma.
- Katta o'zgarishdan oldin qisqa reja berib, tasdiq ol. Xavfsizlik yoki arxitekturaga ta'sir qiladigan noaniq qaror bo'lsa, taxmin qilma, so'ra.
- DB o'zgarishlarini faqat migration orqali qil. API kontraktlarini `docs/API.md` ga yoz.
- Secret va API kalitlarini kodga yozma, `.env` ishlat. Keraksiz dependency qo'shma.
- Faqat design tokenlarni ishlat, tasodifiy rang yoki spacing qo'yma. UI komponentlarini qayta ishlat.
- Fake ball, fake progress yoki demo datani haqiqiy ma'lumot sifatida ko'rsatma. Har bir sahifada loading, empty va error state bo'lsin.
- Owner endpointlari backendda RBAC orqali himoyalanadi. Owner'ning muhim harakatlari audit logga yoziladi.
- Platforma rasmiy SAT score bermaydi, faqat practice performance yoki taxminiy oraliq ko'rsatadi.
- Testlarsiz ish tayyor emas. Tugallanmagan ishni tayyor deb e'lon qilma.
