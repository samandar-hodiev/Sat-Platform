# Progress log

Har bir sessiya ish oxirida shu formatda yozadi (yangi yozuv eng tepaga qo'shiladi):

```
## <raqam>-<sessiya nomi> — YYYY-MM-DD
Status: Done | In progress | Blocked by <raqam>
Bajarildi:
Testlar:
Muammolar:
Keyingi qadam:
```

---

## 2.01-Learner · Profile setup — 2026-10-12
Status: Done
Qarorlar (foydalanuvchi tasdiqladi): DOB to'liq sana, kamida 13 yosh; avatar JPG/PNG/WebP, ≤5 MB, server qayta kodlaydi (≤512×512 JPEG, markazdan kvadrat, EXIF orientation qo'llanadi, metadata tashlanadi); Google akkauntlar uchun parol ixtiyoriy, email-kod akkauntlar uchun majburiy; telefon bitta maydon (default +998), server E.164 ga normallashtiradi.
Bajarildi:
- **Backend** `api/internal/profile/`: `validate.go` (ism, telefon, DOB 13+, IANA timezone), `avatar.go` (tur baytlardan, 40 MP / 12000 px chegarasi, EXIF orientation, crop+scale, JPEG), `store.go`/`repository.go` (`profiles` upsert; setup bitta statement'da `WHERE profile_completed_at IS NULL`; parol bilan bitta tranzaksiyada), `service.go`, `handler.go`, `accounts.go` (`AuthAccounts` adapteri, `GooglePrefiller` → `auth.ProfilePrefiller`), `module.go`.
- Endpointlar (`docs/API.md` → Profile, `openapi.yaml` tag `profile`): `GET /api/v1/me/profile`, `PUT /api/v1/me/profile/setup` (bir marta, keyin 409), `POST|DELETE /api/v1/me/avatar` (multipart `file`, learner'ga soatiga 20), `GET /api/v1/media/avatars/:user/:file` (public, 128-bit tasodifiy URL, immutable cache). Faqat `learner` roli.
- 1.01 bilan kelishib qo'shildi: `auth.Service.Account` (HasGoogle), `SetInitialPasswordTx` + `auth.PgxTx`, `ProfilePrefiller` (Google ismi prefill).
- **Web** `features/profile/`: `api.ts`, `validation.ts`, `profileStore.ts` (`useLearnerProfile` — bitta fetch, setup/shell/gate o'rtasida umumiy), `AvatarPicker`, `ProfileSetupForm`, `ProfileSetup`, `ProfileGate`, `SignOutButton`. Sahifa `app/(setup)/setup/profile` (`(setup)` layout: `SessionBoundary`, brand, theme, Sign out). Loading/error/validatsiya holatlari, email read-only, parol maydonlari `password_required`/`has_password` bo'yicha, timezone brauzerdan.
- Minimal integratsiyalar: `middleware.BodyLimitWithRoutes` (0.00, `BodyLimit` imzosi o'zgarmadi), `server.Deps.Storage` + wiring (`server.go`), `cmd/server/main.go` (`storage.NewLocal`), `lib/api/client.ts` (FormData → multipart), `components/ui/Avatar.tsx` (`xl` o'lcham), `proxy.ts` (`/setup/*` guard), `(learner)/app/layout.tsx` (`ProfileGate`), `learner-shell/account.ts` (profil ismi va avatar), `docs/DB.md` (profiles izohlari).
- Dependency: `golang.org/x/image` (WebP decode, CatmullRom).
Testlar:
- Go: `go vet ./...` toza; `TEST_DATABASE_URL=…/sat_platform_test go test -race -count=1 -p 1 ./...` hammasi o'tdi (profile: validatsiya, avatar, servis, HTTP, PostgreSQL'da 17 senariy — rollback, parallel setup bir marta, parol profil tranzaksiyasida; middleware: per-route body limit).
- Web: `npm test` 18 fayl, 293 test (2.01: 31 ta); `lint`, `typecheck`, `format:check`, `next build` toza.
- Brauzerda (dev, :3000 + :8090): sign-up → `/setup/profile`; `/app` → setup'ga qaytaradi; avatar upload (600×400 → 400×400, ko'rinadi); 13 yoshdan kichik rad etildi; saqlangach `/app`, header'da "Ali Valiyev" va avatar; `/setup/profile` endi `/app` ga o'tkazadi; bazada E.164 telefon, Asia/Tashkent, argon2id parol. 375px mobil (gorizontal scroll yo'q) va dark rejim. Konsolda xato yo'q. Dev bazada test akkauntlar: `profile201.check@example.com`, `profile201.mobile@example.com`.
Muammolar:
- 8090 dagi eski API jarayoni (boshqa sessiya ishga tushirgan) yangi kod bilan qayta ishga tushirildi — dev sessionlar uzildi (`JWT_SECRET` .env da yo'q).
- `AFTER_PROFILE_SETUP` hozircha `/app` — onboarding sahifasi (2.02) yo'q. 2.02 `features/profile/routes.ts` dagi qiymatni o'z yo'liga almashtiradi va kerak bo'lsa `/app` uchun o'z gate'ini qo'shadi.
- Server field xabarlari bo'lak ko'rinishida ("is required"); web ularni gapga aylantiradi. Forma asosan klient validatsiyasiga tayanadi.
- Production storage (S3) yo'q — local driver (5.05). Eski avatar fayli o'chirilmay qolsa (storage xatosi) — faqat warn log, tozalash job'i yo'q.
- Exam (`/exam/*`) layout'ida ProfileGate yo'q — exam'ga faqat `/app` orqali kiriladi.
Keyingi qadam: 2.02 onboarding (`AFTER_PROFILE_SETUP`); 2.04 learner ismini `useLearnerProfile()` (web) yoki `GET /api/v1/me/profile` → `first_name` dan oladi; 2.10 profil tahriri shu modulga `PATCH /me/profile` qo'shadi (setup endpointi qayta ishlatilmaydi), avatar endpointlari tayyor.

## 1.01-Auth · Learner — 2026-10-12
Status: Done. Haqiqiy Google client ID (foydalanuvchi Google Cloud'da yaratadi) va production email provayderi (5.05) — sozlama, kod emas.
Qarorlar (foydalanuvchi tasdiqladi): 15 daqiqalik JWT access token (HS256) va rotating opaque refresh token (`docs/AUTHENTICATION.md`); web'da refresh token HttpOnly cookie'da, mobil'da secure storage'da; email hozircha dev `log` driver. PRD §7 bo'yicha sign-up'da parol so'ralmaydi (2.01 Profile Setup'da).
Bajarildi:
- **Go building blocks** (`api/internal/auth`): `password.go` (Argon2id OWASP, PHC, rehash, timing uchun dummy), `tokens.go` (HS256 JWT, opaque token + SHA-256), `codes.go` (6 raqam, email bilan bog'langan HMAC, TTL 10 daqiqa, 5 urinish, 60 s cooldown, kuniga 10), `google.go` (ID token verifier, JWKS cache, kid rotatsiyasi, Google ishlamay qolsa ma'lum kalit), `email.go`. `platform/mailer` (`Mailer` + dev `Log`, production'da ishlamaydi).
- **Servis va HTTP**: `store.go` (Store/Tx), `service.go` (sign-up start/verify — yangi va mavjud email uchun bir xil javob; sign-in — bir xil xato va timing, email bo'yicha limit; Google — `sub` bo'yicha, birinchi marta email bilan bog'lanadi; refresh rotation + 30 s grace + reuse'da session bekor; logout; forgot/reset — hech narsani oshkor qilmaydi, reset barcha sessionlarni yopadi; me; owner/suspended learner yo'lidan kira olmaydi), `handler.go` (`/api/v1/auth/*`, har birida per-IP limit, `Authenticate` + `CurrentUser`), `module.go`, `repository.go` (PostgreSQL, `FOR UPDATE` lock'lar, noto'g'ri UUID → not found).
- **DB** (migrationlar, `make migrate-up` dev bazaga qo'llandi): `user_identities`, `email_verification_codes`, `auth_sessions` + `auth_refresh_tokens`, `password_reset_tokens`.
- **Config**: `JWT_SECRET` (production'da majburiy, dev'da bo'sh bo'lsa random), `AUTH_*_TTL`, `GOOGLE_CLIENT_IDS`, `MAIL_DRIVER` (`log` | `disabled`, production'da default `disabled`). `.env.example` va `web/.env.example` (`NEXT_PUBLIC_GOOGLE_CLIENT_ID`).
- **Ulash**: `server.Deps.DB` + `server.NewRouter` auth modulini ulaydi, `cmd/server` pool'ni beradi. `openapi.yaml` ga 9 ta auth yo'li (tag `auth`).
- **Web**: sahifalar `/auth/sign-in`, `/auth/sign-up` (email → kod → yangi: `/setup/profile`, mavjud: `/app`), `/auth/forgot-password`, `/auth/reset-password`; faqat 0.02 tokenlari va UI kit. `/api/session/[...action]` (refresh token cookie'da, CSRF: `X-Session-Request` + Origin), `features/auth/session.ts` (xotiradagi access token, single-flight refresh, `withAuth`, `useSession`, `signOut`), `src/proxy.ts` (`/app/*`, `/exam/*` guard, `?next=`), `SessionBoundary`, Google Identity Services tugmasi (client ID bo'lmasa halol "not set up" xabari).
- **Boshqa sessiyalar fayllarida minimal integratsiya**: `(learner)/app/layout.tsx` va `(focus)/exam/layout.tsx` → `SessionBoundary`; `learner-shell/account.ts` → `useShellAccount()` haqiqiy session'dan (ism hozircha email'ning "@" dan oldingi qismi; 2.01 profil ismiga almashtiradi); 0.00 `config_test.go` (production holatiga `JWT_SECRET`); 0.01 `schema_integration_test.go` — "down 1" tekshiruvi jadval nomi o'rniga versiya bo'yicha (har yangi migrationda buzilmasligi uchun, maqsadi o'zgarmadi).
- **2.01 so'rovi bo'yicha (2026-10-12):** `(*Service).SetInitialPassword(ctx, userID, password)` va `SetInitialPasswordTx(ctx, auth.PgxTx(tx), userID, password)` — paroli yo'q akkauntga birinchi parol (policy + Argon2id; parol bor bo'lsa `409 CONFLICT`, user yo'q bo'lsa `404`); `auth.PgxTx(pgx.Tx)` boshqa modul tranzaksiyasiga qo'shilish uchun. Google prefill: `auth.ProfilePrefiller` interfeysi (`PrefillFromGoogle(ctx, userID, GoogleProfile{FirstName, LastName})`) — faqat yangi Google learner uchun, xatosi sign-in'ni to'xtatmaydi (log). Auth `profiles` jadvaliga o'zi yozmaydi (ARCHITECTURE: modul boshqa modul jadvaliga tegmaydi) — 2.01 implementatsiya qiladi, `server.NewRouter` da `auth.Deps.ProfilePrefiller` ga ulanadi. `GoogleIdentity` ga `GivenName`/`FamilyName` qo'shildi. Kirish usullari: `(*Service).Account(ctx, userID) (Account, error)` — `Account{User, HasGoogle}`, `HasPassword()`; `GET /auth/me` endi `user.has_google` ham qaytaradi.
- Hujjatlar: `docs/API.md` → "Auth — learner (1.01)", `docs/DB.md` → ERD, "Auth tables", migration log, `docs/AUTHENTICATION.md` → "Web session as built".
Testlar: Go — `go vet`, gofmt toza; `TEST_DATABASE_URL=…/sat_platform_test go test -race -count=1 -p 1 ./...` hammasi o'tdi (auth: unit + HTTP + 20 ta servis senariysi haqiqiy PostgreSQL'da, suspended/deleted, session client info). Web — `npm test` 15 fayl, 262 test (auth: 64), `lint`, `typecheck`, `format:check`, `next build` toza. Brauzerda to'liq oqim: `/app/practice` → sign-in (`?next=`); sign-up → kod (dev log) → `/setup/profile`; `/app` session'ni cookie'dan tiklaydi, header'da `e2e.learner`; Sign out → bazada session `revoked_at`, `/app` yana sign-in'ga. Header'siz session so'rovi 403. Dev bazada test akkaunt qoldi: `e2e.learner@example.com`.
Muammolar:
- Production'da `TRUSTED_PROXIES` ga web server kiritilishi kerak (5.05), aks holda web foydalanuvchilar bitta per-IP limitni bo'lishadi.
- Production email provayderi yo'q — `MAIL_DRIVER=disabled` da sign-up kodi va reset 503 (5.05).
- Refresh token reuse'da audit log (1.02 helper) va xavfsizlik bildirishnomasi (3.14) yo'q — hozircha warn log.
- Eskirgan kodlar/tokenlar/sessionlar tozalanmaydi (cleanup job — 5.05).
- `JWT_SECRET` dev `.env` da yo'q — API har restartda sessionlarni tugatadi. Doimiy sessiya kerak bo'lsa `.env` ga `openssl rand -hex 32` qiymatini qo'shing.
- 2.01 diqqatiga: `/setup/profile` yo'li `web/src/features/auth/routes.ts` da; parolni o'rnatish uchun API kerak bo'lsa (`has_password: false` foydalanuvchilar) — 1.01 ga so'rov yuboring.
Keyingi qadam: Google Cloud'da OAuth client (web, keyin Android/iOS) yaratib `GOOGLE_CLIENT_IDS` va `NEXT_PUBLIC_GOOGLE_CLIENT_ID` ni to'ldirish; 1.02 owner auth shu modul ustiga quriladi (`Authenticate`, sessions jadvali); boshqa modullar learner route'larini `authModule.Authenticate()` bilan himoyalaydi.
## 0.01-Foundation · Database & Migrations — 2026-10-12
Status: Done
Qarorlar (foydalanuvchi tasdiqladi): golang-migrate kutubxona sifatida, `cmd/migrate` ichiga embed qilingan. API ishga tushganda migration avtomatik ishlamaydi, u alohida qadam.
Bajarildi:
- Migrationlar (`api/migrations/`, har biri `BEGIN…COMMIT` va aniq `down` bilan):
  - `set_updated_at()` umumiy trigger funksiyasi.
  - `users`: lowercase email CHECK, `deleted_at IS NULL` bo'yicha partial unique, `password_hash` NULL bo'lishi mumkin, `role` learner/owner, `status` active/suspended, `(role, created_at)` indeksi.
  - `profiles`: PK `user_id` cascade, E.164 phone, timezone, `profile_completed_at`.
  - `user_preferences`: theme/accent/density/motion 0.02 qiymatlari bilan, `sidebar_collapsed`.
  - `audit_logs`: append-only trigger UPDATE/DELETE/TRUNCATE ni rad etadi (42501). Faqat user o'chirilganda `actor_user_id → NULL` ga ruxsat bor. `entity.verb` action CHECK, JSONB object metadata, 4 ta filter indeksi.
- Go: `internal/platform/migrate` (Up/Down/DownAll/Force/Status, fayl nomlari va up/down juftlarini tekshirish, URL xatoda chiqmaydi), `cmd/migrate` (`up | down [n] | status | force | create <name>`), `migrations/embed.go`, `config.DatabaseURL()` (faqat `DATABASE_URL`, JWT va boshqa sozlamalarsiz).
- Makefile: `migrate-up`, `migrate-down [n=]`, `migrate-status`, `migrate-create name=`.
- Hujjatlar: `docs/DB.md` (tool, buyruqlar, mermaid ERD, jadvallar tavsifi, migration log), `ARCHITECTURE.md` (layout, decision #9), `DEVELOPMENT.md`, `api/migrations/README.md`.
- Dependency: `github.com/golang-migrate/migrate/v4` v4.20.1.
- Lokal `sat_platform` bazasiga `make migrate-up` qo'llandi (5 ta migration). Qayta `up` hech narsa o'zgartirmaydi.
Testlar: `go test -race` va `TEST_DATABASE_URL` (sat_platform_test) bilan quyidagilar o'tdi:
- `cmd/migrate`: create va argumentlar tekshiruvi.
- `internal/platform/migrate`: fayl validatsiyasi, BEGIN/COMMIT, up → down 1 → down all → up, qayta `up` ma'lumotni saqlashi.
- Schema constraintlari (11 subtest): email normallashtirish, role/status, soft-delete'dan keyin emailni qayta ishlatish, updated_at trigger, profiles/preferences CHECKlar, cascade, audit append-only va SET NULL.
- `internal/config` o'tdi. 0.01 paketlarida `go vet` va gofmt toza.
- To'liq `go test ./...` da faqat `internal/auth` build xatosi bor (`NewCode redeclared`, `store.go`). Bu 1.01 sessiyasining hozir yozilayotgan kodi, 0.01 ga aloqasi yo'q.
Muammolar: `profiles.timezone` IANA nomini DB tekshirmaydi (CHECK subquery qila olmaydi), API `time.LoadLocation` bilan tekshiradi. Retention purge kerak bo'lsa (`audit_logs`), alohida migration va qaror kerak. Commit qilinmadi.
Keyingi qadam:
- 1.01 B bosqichi `users` ga tayanadi. Auth jadvallari `make migrate-create name=create_auth_sessions` bilan qo'shiladi.
- 2.01 → `profiles` (upsert), 2.11 → `user_preferences` va wallpaper/notification ustunlari, 1.02 → audit helper va owner seed.
- Har bir yangi jadval uchun `docs/DB.md` dagi ERD va jadvallar ro'yxati yangilanadi.

## 2.04-Learner · Dashboard Home — 2026-10-12
Status: Blocked by har bir blokning manba sessiyasi (2.02, 2.05, 2.07, 2.08, 2.09, 3.02, 3.04, 3.06, 3.08, 3.09, 3.11, 3.14)
Bajarildi: Faqat audit, kod o'zgartirilmadi.
- Mavjud dashboard: `web/src/app/(learner)/app/page.tsx` — 2.03 qoldirgan vaqtinchalik sarlavha ("Your study overview will appear here."). API chaqiruvi, blok yoki state yo'q.
- Backend: `api/migrations/` bo'sh (0.01 bajarilmagan, `users`/`profiles`/`user_preferences` yo'q). `docs/API.md` da faqat `/health`, `/ready`, `/openapi.yaml`. `/auth/me` va learner session yo'q (1.01 B). `/app` route guard'siz.
- Foydalanish mumkin: 2.03 shell (`LearnerShell`, header actions slot, `nav.ts` yo'llari) va 0.02 UI kit (`Card`, `Badge`, `Skeleton`, `LoadingState`/`EmptyState`/`ErrorState`, `Button`).
- Topshiriqdagi bloklar va manbalar (hammasi hozircha yo'q):
  - A. SAT Goal (target, imtihon sanasi, qolgan kunlar) — 2.02; oxirgi mock — 3.08/3.09.
  - B. Today's Study Plan (vazifalar, davomiylik, holat, kunlik progress) — 2.05.
  - C. Math / R&W progress, domain kuchli/zaif, accuracy, taqqoslash — 2.07, 3.02 (domainlar), 3.06 (practice), 3.11 (diagnostika).
  - D. Continue Learning — 3.04 (lesson progress), 3.06 (practice set); "birinchi mavzu" tavsiyasi — 3.02/3.11.
  - E. Recent Mock Exam + Review CTA — 3.08, 3.09.
  - F. Recommended Next Steps — 2.07, 2.09, 3.11 ma'lumotlari.
  - Shortcutlar: Saved (2.08), Mistakes (2.09), Library (3.10), AI (2.06) — faqat havola, sahifalar hozir shell'dagi "Page not found" ga olib boradi. Streak va haftalik activity — 2.07. Notifications — 3.14 (header slot).
- Xulosa: hech bir blok uchun haqiqiy ma'lumot manbasi yo'q. Hozir yozilsa, yo o'ylab topilgan API kontraktiga, yo soxta ma'lumotga tayanadi (CLAUDE.md qoidasi 4 ga zid). "Yangi learner" va "ma'lumot yo'q" holatini ajratish ham `/auth/me` siz mumkin emas.
Testlar: — (kod o'zgarmadi)
Muammolar: Mobil (Android/iOS) ham ishlatishi uchun dashboard API'si additive va bitta aggregatsiya endpointi bo'lishi ma'qul — manba modullar tayyor bo'lgach belgilanadi.
Keyingi qadam (tavsiya etilgan tartibda 2.07 dan keyin):
1. `GET /api/v1/learner/dashboard` — bitta aggregatsiya endpointi (`api/internal/dashboard/`), manba modullarning servis interfeyslarini o'qiydi, o'z jadvali yo'q. Har bir blok alohida `status: ready | empty | unavailable` bilan qaytadi — bitta modul xatosi butun sahifani yiqitmaydi. Ballar faqat "estimated practice range" sifatida, rasmiy SAT natijasi bilan aralashtirilmaydi.
2. Web: `features/learner-dashboard/` — ierarxiya: Goal + Today's Plan (yuqorida), Continue Learning + Recent Mock, Progress, Next Steps; shortcutlar ixcham qatorda; streak va notifications header slot'ida. Har blok uchun loading/empty/error/ready, retry.
3. Tasdiq talab qiladigan qarorlar: streak qoidasi (timezone, minimal faollik), weak topic tanlash (minimal savollar soni, nechta topic), tavsiyalar qoidalari va "yetarli ma'lumot" chegarasi.
4. Testlar: Go handler/servis testlari (har blok holati), web komponent testlari (4 holat × blok), brauzerda light/dark, wallpaper, 375/768/1280px.

## 2.03-Learner · Dashboard UI — 2026-10-12
Status: Done (layout). Route guard va header'dagi haqiqiy user/logout — Blocked by 1.01 (B bosqich: web session, `/auth/me`, `/auth/logout`)
Qarorlar (foydalanuvchi tasdiqladi): sidebar holati cookie'da (`sat_learner_sidebar`); mobil'da drawer (bottom nav yo'q); hali qurilmagan sahifalar sidebar'da ko'rinadi va shell ichidagi "Page not found" ga olib boradi; learner route'lari `/app/*`, exam runner `/exam/*` (`(focus)` guruhi).
Bajarildi:
- `web/src/features/learner-shell/`: `nav.ts` (PRD §10.2 dagi 12 ta havola, Learn/Track/Account; `activeNavItem` — nested sahifalar o'z bo'limini belgilaydi, Dashboard boshqalarni "egallamaydi"), `sidebarState.ts` (cookie), `account.ts` (`useShellAccount` — `loading | unavailable | ready`), `icons.tsx` (o'z ikonkalari, 0.02 kit uslubida), komponentlar: `LearnerShell`, `Sidebar` (yig'iladi, faqat ikonkalar + `title`, screen reader uchun label saqlanadi), `SidebarNav` (`aria-current`, guruhlar labelled list), `LearnerHeader` (breadcrumb, actions slot, `UserMenu`), `MobileDrawer` (native `<dialog>`: focus containment, Escape, backdrop, havola tanlanganda yopiladi), `UserMenu` (Avatar + DropdownMenu: My Profile, Settings, Subscription, Sign out), `Brand`.
- Route'lar: `app/(learner)/app/layout.tsx` (server'da cookie o'qiydi — birinchi paint'da sakrash yo'q), `/app` (vaqtinchalik sarlavha, 2.04 almashtiradi), `[...missing]` + `not-found.tsx`, `error.tsx`, `loading.tsx` (shell ichida). `app/(focus)/exam/layout.tsx` (sidebar/header'siz) + not-found. `/app` va `/exam` `noindex`.
- Skip link, `banner`/`main`/`navigation` landmarklar, header sticky, faqat 0.02 tokenlari va UI kit.
- Fake ma'lumot yo'q: session bo'lmagani uchun user menyu "Account details are unavailable" deydi, Sign out disabled. Notifications va streak ko'rsatilmaydi (faqat kodda slot).
- `docs/ARCHITECTURE.md` §4.2 Learner shell, route jadvalida `(focus)` = `/exam/*`.
Testlar: `npm test` — 12 fayl, 239 test o'tdi (2.03: 36 ta — nav konfiguratsiyasi va active matching, cookie parse/serialize, landmarklar, skip link, guruhlar, aria-current, breadcrumb, collapse + cookie, server'dan collapsed holat, drawer ochish/Escape/havola/close tugmasi, user menyu loading/unavailable/ready, navigatsiya, sign out va uning xatosi). `lint`, `typecheck`, `prettier --check`, `next build` — toza. Brauzerda: desktop light/dark, collapse reload'dan keyin saqlanadi, 375px mobil drawer, user menyu, `/app/exams/abc` va `/exam/x` 404 o'z layout'larida, konsolda xato yo'q.
Muammolar:
- Route guard yo'q: `/app` ni hozir har kim ochishi mumkin (ma'lumotni baribir API himoya qiladi). 1.01 B bosqichi web session'ni qo'shganda `(learner)/app/layout.tsx` ga guard va `useShellAccount()` ga haqiqiy session ulanadi (boshqa komponentlar o'zgarmaydi).
- 1.01 eslatmasi: `AFTER_SIGN_IN = "/app"` — mos keladi, o'zgartirish shart emas.
- Header search (PRD §10.3 "zarur bo'lsa") qo'yilmadi — qaysi sahifalar uchun kerakligi aniq emas.
Keyingi qadam: 1.01 B — `useShellAccount()` va route guard. Sahifa sessiyalari (2.04–2.11, 3.xx) `nav.ts` dagi yo'lda `page.tsx` yaratadi. 3.14 va 2.04/2.07 header actions slot'iga o'z indikatorlarini qo'shadi. 3.08 `/exam/[attemptId]` ni `(focus)` ichida quradi. 2.11 sidebar cookie'sini account preferences bilan sinxronlashi mumkin.

## 0.02-Foundation · Design system & Themes — 2026-10-11
Status: Done
Qarorlar (foydalanuvchi tasdiqladi): minimalistik, neytral dizayn; Radix va ikonka kutubxonasi yo'q (komponentlar va ikonkalar o'zimizniki); shrift Inter (`next/font`). Theme sozlamasi cookie'da saqlanadi, chunki ESLint `localStorage` ni taqiqlaydi.
Bajarildi:
- Tokenlar (`web/src/app/globals.css`): neytral light/dark palitra (OKLCH), primary/secondary, surface, text, border, success/warning/error/info (`-soft`, `-text`, `error-fg`), chart ranglari, typography scale, radius, shadow, z-index, duration/easing, control height. Tailwind default palitrasi o'chirilgan, shuning uchun faqat tokenlar ishlaydi.
- Theme tizimi: Light/Dark/System, 6 ta accent (neutral default, blue, indigo, violet, teal, rose), density (comfortable/compact), motion (system/reduced/full) uchun CSS ilgaklari. Pre-paint inline script (miltillash yo'q). `ThemeProvider` + `useTheme()`. Learner va Owner sozlamalari alohida cookie'da. `ThemeSwitcher`.
- UI kit (`web/src/components/ui`): Button, IconButton, FormField/Label, Input, Textarea, Select, Checkbox, Radio, ChoiceGroup, Switch, Card, Badge, Avatar, Tabs, Accordion, Modal (native dialog), DropdownMenu, Tooltip, Toast, Spinner, Skeleton, LoadingState, EmptyState, ErrorState, icon set. Klaviatura va ARIA pattern'lari, focus-visible, disabled va loading holatlari.
- Umumiy `app/loading.tsx`, `error.tsx`, `not-found.tsx` tokenlar bilan qayta bezatildi. Root layout'ga Inter, theme script va provider'lar (Theme, Toast) ulandi.
- `/dev/ui-kit` katalogi faqat development'da ochiladi. Production'da `/dev/*` 404 qaytaradi (`next.config.ts` rewrite, minimal integratsiya).
- Dev-dependency'lar: `@testing-library/react`, `@testing-library/user-event`, `@testing-library/dom`, `jsdom`. Vitest setup: `src/test/setup.ts`.
- `docs/ARCHITECTURE.md` §4.1 — tokenlar va komponentlardan foydalanish qoidalari.
Testlar: `npm test` — 6 fayl, 162 test o'tdi:
- Tokenlar: har bir theme va accent uchun WCAG AA kontrast (matnda 4.5:1, ring/primary 3:1) va har bir accentda to'liq token to'plami.
- Theme: cookie parse/serialize, buzilgan qiymatlar, scope'lar, inline script `theme.ts` bilan bir xil natija beradi.
- ThemeProvider va komponentlar: klaviatura navigatsiyasi, ARIA, modal, menu, tooltip, toast, avatar fallback.
`npm run typecheck`, `npm run lint`, `format:check`, `npm run build` — toza. Production'da `curl`: `/dev/ui-kit` → 404, `/` → 200. Browser'da light/dark, accent, reload'dan keyin saqlanish, modal, dropdown va 375px mobil kenglik tekshirildi, console'da xato yo'q.
Muammolar:
- `make web` preview tool orqali "getcwd: Operation not permitted" bilan to'xtadi (sandbox). `npm run dev` to'g'ridan-to'g'ri ishlaydi. `.claude/launch.json` o'zgartirilmadi.
- `app/loading.tsx` borligi sababli page ichidagi `notFound()` 404 emas, 200 status bilan stream qiladi (Next.js xatti-harakati). Sahifa SEO uchun 404 bo'lishi kerak bo'lsa, bu e'tiborga olinsin.
- Input chegaralari (`border-strong`) 3:1 kontrastdan past, minimalistik ko'rinish uchun shunday qoldirildi. Focus holati esa 3:1 dan yuqori. 5.03 a11y review'da qayta ko'rilsin.
Keyingi qadam: 2.03 va 4.01 layout'lari, 0.03 landing va 1.01 auth sahifalari shu kitdan foydalanadi. 2.11 Appearance sahifasini `useTheme().setPreferences` ga ulaydi (DB sync, wallpaper).

## 0.00-Foundation · Architecture & Repo setup — 2026-10-11
Status: Done
Bajarildi:
- Repo: git init, `.gitignore`, `.editorconfig`, root `.env.example` (API) va `web/.env.example` (faqat `NEXT_PUBLIC_*`), `Makefile` (`setup`, `api`, `web`, `test`, `test-api-integration`, `lint`, `fmt`, `typecheck`, `build`, `gitpulse-hook`), GitPulse pre-push hook (`infrastructure/scripts`).
- `api/` (Go 1.26, Gin, pgx): `cmd/server` (graceful shutdown), `internal/config` (env + validatsiya, barcha xatolar birdan, production qat'iyroq), `platform/logger` (slog), `apperr` + `httpx` (yagona envelope, 422 field xatolari, pagination), `middleware` (request id, access log, recovery, security headers, CORS — faqat aniq originlar, body limit, error handler, rate limit), `platform/ratelimit` (interfeys + in-memory), `platform/storage` (ObjectStorage interfeysi + local, path traversal himoyasi), `platform/database` (pgx pool), `health` (`GET /health` liveness, `GET /ready` DB readiness, 503), `openapi/openapi.yaml` (`GET /api/v1/openapi.yaml`).
- `web/` (Next.js 16.3.8, React 19, TS strict, Tailwind 4, ESLint, Prettier, Vitest): root layout, vaqtinchalik bosh sahifa, umumiy `loading`/`error`/`not-found`, `src/lib/api/client.ts` (yagona API client, `ApiError`), `src/types/api.ts`, `src/lib/env.ts`, security headerlar.
- Hujjatlar: `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` (qoidalar; ERD — 0.01), `docs/AUTHENTICATION.md` (web + mobil token arxitekturasi; 1.01 tasdiqlaydi), `docs/DEVELOPMENT.md`, `README.md`, `mobile/README.md` (Flutter).
Testlar: `go test ./...` — barcha paketlar o'tdi; `TestPostgresIntegration` haqiqiy `sat_platform_test` bazasida o'tdi; `go vet`, gofmt toza. Web: `npm run lint`, `typecheck`, `test` (6 test), `prettier --check`, `npm run build` — o'tdi. Jonli: `/health` 200, `/ready` 200 (DB bilan) va 503 (DB'siz), begona origin preflight 403, SIGTERM'da graceful shutdown, yetishmayotgan config'da tushunarli xato.
Muammolar: `npm audit` — `eslint-config-next` zanjirida 5 ta high (faqat dev vositasi; tuzatish faqat Next 14 ga tushirish, mos emas). Rate limiter in-memory — bir nechta API instance uchun umumiy store kerak (5.05). Docker yo'q, lokal Postgres.app ishlatildi. Papka nomida bo'sh joy bor va `~/Desktop` iCloud'da — `node_modules` sinxronlanadi.
Keyingi qadam: 0.01 (migration tool, `users`/`profiles`/`user_preferences`/`audit_logs`, ERD), 0.02 (design tokenlar, `web/src/components`). Yangi modul: `api/internal/<module>/` + `server.NewRouter` da `RegisterRoutes(v1)` + `docs/API.md` va `openapi.yaml`.

## 5.05-Production · Deploy, Monitoring & Backups — 2026-10-11
Status: Blocked by 0.00, 0.01 (CI'dagi to'liq pipeline uchun qisman 5.04; prod oldidan 5.02 va 5.01)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor. `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali yo'q. Dockerfile, docker-compose, CI konfiguratsiyasi va `.env.example` ham yo'q. Repo hali git repository emas. PRD §3, §28 (database backups), §33, §34 (Phase 5) va §35 o'qildi. PRD'da deploy, monitoring va backup tafsilotlari yozilmagan.
Testlar: —
Muammolar: CI/CD, deploy, monitoring va backuplar quyidagilarga tayanadi:
- 0.00 (asosiy bloker): monorepo tuzilmasi, Go build entrypoint, config va `.env` kalitlari ro'yxati, structured logger (log formati monitoring uchun), healthcheck/readiness endpointi uchun joy, storage interfeysi (upload fayllari qayerda saqlanadi va backupga kiradimi), `web/` build sozlamalari. Bularsiz Dockerfile, CI qadamlari va env ro'yxatini yozib bo'lmaydi.
- 0.01: migration tool va uni ishga tushirish buyrug'i (deploy vaqtidagi migration qadami), PostgreSQL versiyasi va ulanish konfiguratsiyasi (backup/restore skriptlari uchun).
- 5.04 (qisman): CI'da E2E testlarni ishga tushirish. 5.04 tayyor bo'lmaguncha CI faqat lint, unit va integration testlarni ishlatadi.
- 5.02 va 5.01 (qisman): production'ga chiqishdan oldin security review (secretlar, CORS, headerlar) va haqiqiy to'lov webhooklari uchun ochiq URL va secretlar.
- Eslatma: build va test buyruqlari hali aniqlanmagan, shuning uchun pipeline'ni bo'sh repo uchun yozib bo'lmaydi. Vaqtinchalik fake konfiguratsiya yozilmadi.
Keyingi qadam: 0.00 va 0.01 tugagach (tavsiya etilgan tartibda oxirgi sessiya, 5.01 dan keyin) 5.05 ni boshlash. Avval reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi:
- Hosting: managed (masalan web uchun Vercel, api uchun Fly.io/Render/Railway, managed Postgres) yoki bitta VPS'da Docker Compose + reverse proxy (Caddy/Nginx, avtomatik TLS). Region va budjet.
- Git hosting va CI: GitHub Actions (tavsiya). Repo hali git emas, uni boshlash kerak. Branch strategiyasi: `main` → production, staging muhiti bo'ladimi.
- Deploy oqimi: Docker image registry, migrationlar deploydan oldin alohida qadamda ishlaydi, rollback usuli, zero-downtime talab qilinadimi.
- Secretlar: CI/hosting secret store. `.env.example` faqat kalit nomlari bilan, qiymatlarsiz.
- Monitoring: error tracking (masalan Sentry, yangi dependency, tasdiq kerak), uptime check (`/healthz`, `/readyz`), log yig'ish va saqlash muddati, alertlar qayerga boradi (email/Telegram), asosiy metrikalar (latency, 5xx, DB ulanishlari).
- Backuplar: managed Postgres PITR yoki `pg_dump` cron. Chastota, saqlash muddati (retention), shifrlash, offsite joy (S3-mos storage), upload fayllari backupi. Restore'ni muntazam sinash va restore runbook'i (`docs/` ga).
- Testlar: CI pipeline'ning o'zi (lint, `go test`, `web` testlari, build), staging'ga deploydan keyin smoke test, backup → restore sinovi.

## 5.04-Production · E2E tests — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 1.02, 2.01, 2.02, 2.03, 2.11, 3.01, 3.02, 3.05, 3.06, 3.07, 3.08, 3.09, 3.11, 3.12, 3.13, 4.01, 4.05
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PROGRESS.md'dagi barcha oldingi sessiyalar "Blocked" holatida. PRD §33 o'qildi.
Testlar: —
Muammolar: E2E testlar ishlaydigan ilovaga (web + api + DB) qarshi yoziladi; selektorlar, route'lar va API kontraktlarisiz test yozish fake implementatsiya bo'ladi (CLAUDE.md qoida 4). §33 dagi 15 ta flow va ularning manbasi:
- 1. Registration, 2. Email verification, 3. Login/logout — 1.01 (email kodini testda olish usuli — mail catcher yoki test-only endpoint — 1.01/0.00 bilan kelishiladi).
- 4. Profile setup — 2.01. 5. Onboarding — 2.02. 6. Placement test — 3.11.
- 7. Daily practice — 3.06 (savollar 3.05). 8. Mock exam — 3.07/3.08. 9. Exam results — 3.09.
- 10. Learner settings — 2.11. 11. Owner login — 1.02 (MFA testda qanday o'tiladi — TOTP test secret).
- 12. CMS content creation, 14. Content publishing — 3.01, 3.02, 3.03/3.05. 13. AI draft creation — 4.05, 3.12 (testda AI provider stub rejimi kerak).
- 15. Learner access restrictions — 1.01/1.02 RBAC, 3.13 feature gating.
- Umumiy: layoutlar 2.03, 4.01; UI kit 0.02; repo tuzilmasi va test infratuzilmasi joyi 0.00.
Keyingi qadam: yuqoridagi sessiyalar Done bo'lgach 5.04 ni boshlash. Boshlashdan oldin reja beriladi va tasdiq olinadi: E2E framework (Playwright tavsiya etiladi — `web/` ichida yoki alohida `e2e/` papkada, 0.00 tuzilmasiga mos); test muhiti (docker-compose bilan alohida test DB, har run oldidan migration + seed, testlar orasida izolyatsiya); test-only seed/fixture'lar (owner akkaunt, published curriculum, savollar, mock exam) faqat test muhitida; email va AI uchun stub; CI'da ishga tushirish (5.05 bilan bog'lanadi); barqaror selektorlar uchun `data-testid` konvensiyasi — buni hozirdan barcha frontend sessiyalari qo'llashi foydali.

## 5.02-Production · Security review — 2026-10-11
Status: Blocked by 0.00, 0.01, 1.01, 1.02 (to'liq review uchun — barcha 2.xx, 3.xx, 4.xx modullar va 5.01)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §28 o'qildi. Ko'rib chiqiladigan kod yo'q — security review bo'sh repo ustida o'tkazilmaydi.
Testlar: —
Muammolar: 5.02 boshqa sessiyalar yozgan kodni tekshiradi va topilgan zaifliklarni tuzatadi (yangi feature yasamaydi). Review doirasi (§28) va har bir punkt qaysi sessiya kodiga tayanadi:
- CORS, security headerlar (CSP, HSTS, X-Frame-Options, Referrer-Policy), rate limiter, error handler (stack trace/ichki xato chiqmasligi), config va secretlar (`.env`, kodda kalit yo'qligi) — 0.00.
- SQL injection (parametrlangan so'rovlar), migrationlardagi constraintlar, `audit_logs` immutability — 0.01 va barcha modullar.
- Password hashing, session/token boshqaruvi (httpOnly, Secure, SameSite cookie, rotation, revoke), CSRF himoyasi, login/email code/reset brute-force limitlari, OAuth state/PKCE — 1.01.
- Owner MFA, RBAC middleware barcha `/owner` endpointlarida, audit helper — 1.02.
- IDOR / "learner faqat o'z ma'lumotlari" (har bir learner endpointi user_id bo'yicha filtrlaydimi) — 2.xx, 3.xx.
- XSS: lesson/question/formula kontentidagi rich text, markdown, LaTeX rendering va sanitizatsiya — 3.01, 3.03, 3.05, 3.10; Next.js'da `dangerouslySetInnerHTML` ishlatilishi.
- File upload validatsiyasi (MIME/magic bytes, hajm, nom, EXIF, storage'dan to'g'ridan-to'g'ri bajarilmasligi) — 2.01 avatar, 2.11 wallpaper, 0.00 storage interfeysi.
- Exam yaxlitligi (to'g'ri javoblar submitdan oldin clientga chiqmasligi, server-side timer, idempotent submit) — 3.07, 3.08, 3.11.
- AI: prompt injection, kalitlar frontendga chiqmasligi, usage limit va cost cap — 3.12, 2.06, 4.05.
- Feature gating faqat backendda — 3.13; webhook imzo tekshiruvi va replay himoyasi — 5.01.
- Account deletion va data privacy (cascade, loglarda PII maskalash) — 2.10, 4.07.
- Dependency audit (Go uchun `govulncheck`, `web` uchun `npm audit`) — barcha.
Keyingi qadam: tavsiya etilgan tartibda 5.01 dan keyin (barcha modullar tugagach) 5.02 ni boshlash. Minimal holatda 0.00 → 0.01 → 1.01 → 1.02 tugagach foundation va auth uchun dastlabki review o'tkazish mumkin. Avval reja beriladi va tasdiq olinadi: review formati (topilmalar severity bilan `docs/SECURITY.md` da); kichik tuzatishlarni 5.02 o'zi qiladimi yoki modul sessiyasiga yo'naltiradimi (CLAUDE.md qoidasiga ko'ra umumiy middleware va headerlar — 5.02, modul logikasi — o'sha sessiya); CSRF strategiyasi (SameSite + double-submit token vs Origin tekshiruvi); CSP siyosati (nonce asosida); avtomatik tekshiruvlar CI'ga qo'shiladimi (5.05 bilan kelishiladi); har bir topilma uchun regression test.

## 5.03-Production · Performance & Accessibility — 2026-10-11
Status: Blocked by 0.00, 0.02, 2.03, 4.01 (to'liq review uchun amalda barcha 0.xx–4.xx feature sessiyalari, ayniqsa 3.08)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PROGRESS'dagi barcha oldingi yozuvlar "Blocked" holatida. PRD §29, §30, §32, §4 o'qildi.
Testlar: —
Muammolar: 5.03 — mavjud kodni ko'rib chiqib tuzatadigan review sessiyasi; review qilinadigan kod hali yo'q. Tayanchlar:
- 0.00: `web/` (Next.js) va `api/` tuzilmasi, build/test tooling — Lighthouse/axe va bundle tahlilini ulash joyi.
- 0.02: design tokenlar (kontrast tekshiruvi light/dark/accent kombinatsiyalarida), focus-visible uslubi, `prefers-reduced-motion`, UI kit komponentlarining a11y semantikasi.
- 2.03 / 4.01: learner va owner layoutlari — responsive sidebar/mobile drawer, skip link, landmarklar, klaviatura bilan navigatsiya.
- 3.08: mock exam runner kichik ekranda ishlashi, timer va navigator'ning screen reader e'lonlari, qurilma tavsiyasi (§29).
- Qolgan feature sahifalari: loading/empty/error/success statelar (§32), formalar label va validatsiya xabarlari (§30).
5.03 boshqa sessiyalarning funksiyasini qayta yozmaydi; faqat performance/a11y tuzatishlari va tekshiruv tooling'ini qo'shadi.
Keyingi qadam: feature sessiyalari (kamida 0.00, 0.02, 2.03, 4.01, 0.03 va asosiy 2.xx/3.xx sahifalar) tugagach 5.03 ni boshlash. Avval reja beriladi va tasdiq olinadi: maqsadli ko'rsatkichlar (Core Web Vitals: LCP, INP, CLS; Lighthouse performance/a11y ball chegaralari) va qaysi sahifalarda o'lchanadi; WCAG darajasi (2.2 AA tavsiya etiladi); avtomatik tekshiruvlar (`@axe-core/playwright` yoki jest-axe, Lighthouse CI — yangi dependency sifatida tasdiq bilan) va ularni 5.05 CI'ga ulash; backend tomoni (sekin so'rovlar, indekslar, N+1, response kesh/gzip, pagination); qo'lda tekshiruv ro'yxati (klaviatura, VoiceOver/NVDA, 200% zoom, 320px kenglik, reduce motion).

## 5.01-Production · Payment integration — 2026-10-11
Status: Blocked by 3.13 (va u orqali 0.00, 0.01, 0.02, 1.01, 1.02, 2.03, 4.01; payment failure bildirishnomalari uchun qisman 3.14)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §27, §20 (Subscription), §28 va PROGRESS'dagi 3.13 yozuvi o'qildi.
Testlar: —
Muammolar: Payment integration (haqiqiy to'lov provayderini ulash: checkout/to'lov sessiyasi yaratish, webhooklarni qabul qilish va imzosini tekshirish, idempotent qayta ishlash, renewal, cancellation, payment failure → subscription holatini yangilash; payment status faqat backendda provayder webhook/API orqali belgilanadi, frontend ma'lumotiga qarab emas — §27) quyidagilarga tayanadi:
- 3.13 (asosiy bloker): plan definitions, `subscriptions` va `payments` jadvallari, `entitlements` servisi, subscription status mashinasi va `PaymentProvider` interfeysi. 5.01 shu interfeysning haqiqiy implementatsiyasini yozadi — subscription logikasini qayta yasamaydi.
- 0.00: Go core (config — provayder kalitlari va webhook secret faqat `.env`da, error handler, logger, middleware, rate limiter), `api/internal/<module>/` tuzilmasi. 0.01: migration tool, `users`, `audit_logs` (webhook event loglari/`payment_events` jadvalini 5.01 o'z migrationi bilan qo'shadi).
- 1.01: learner auth (checkout faqat login qilgan learner uchun). 1.02: owner RBAC va audit helper (owner tomonidan refund/qo'lda o'zgartirish audit logga yoziladi).
- 0.02, 2.03, 4.01: Learner Settings → Subscription tab va Owner subscriptions sahifasidagi checkout/billing tugmalari uchun UI kit va layoutlar (sahifalarning o'zi 3.13 ga tegishli; 5.01 faqat checkout oqimini ulaydi).
- 3.14 (qisman): payment failure / renewal bildirishnomalari 3.14 emit API'si orqali yuboriladi.
Keyingi qadam: tavsiya etilgan tartibda 4.07 dan keyin (3.13 tugagach) 5.01 ni boshlash. Avval reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi: to'lov provayderi (masalan Stripe vs mahalliy Payme/Click — valyuta va target bozor; bir nechta provayder kerakmi); hosted checkout vs embedded form (karta ma'lumotlari serverimizga tegmasligi kerak — PCI); yangi dependency (rasmiy Go SDK vs `net/http`) — tasdiq kerak; webhook imzo tekshiruvi, replay himoyasi va idempotency (provider event ID bo'yicha unique constraint); renewal va proration; refundlar Owner panelidan qilinadimi; test/sandbox rejimi va kalitlarni ajratish; testlar: imzo, idempotency, holat o'tishlari uchun unit va fake provider bilan integratsiya testlari.

## 4.06-Owner · Platform Settings — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 4.01
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §21 (21.1, 21.3), §2 (Owner funksiyalari ro'yxati), §28 o'qildi. PRD'da Platform Settings tarkibi batafsil yozilmagan (faqat §2 va §21.3 da nomi bor).
Testlar: —
Muammolar: Platform Settings sahifasi (`/owner/settings`) quyidagilarga tayanadi:
- 0.00: Go core (config, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi. 0.01: migration tool, `audit_logs`; sozlamalar jadvalini (masalan `platform_settings`) 4.06 o'z migrationi bilan qo'shadi. 0.02: UI kit (form, input, switch, select, toast, empty/error state). 1.02: owner RBAC middleware va audit helper — har bir sozlama o'zgarishi audit logga yoziladi (eski/yangi qiymat). 4.01: owner layout va sidebar'dagi "Platform Settings" havolasi.
- Boshqa modullarning o'z sozlamalari (AI limitlari va cost — 3.12, planlar/feature gating — 3.13, bildirishnoma shablonlari — 3.14, owner theme — 4.01) 4.06 da qayta yasalmaydi; kerak bo'lsa faqat havola yoki o'qish.
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.02 → 4.01 tugagach (tavsiya etilgan tartibda 4.04 dan keyin) 4.06 ni boshlash. PRD'da tarkib yo'qligi sababli avval reja beriladi va quyidagilar bo'yicha tasdiq olinadi: sahifadagi sozlamalar ro'yxati (masalan platforma nomi/support email, ro'yxatdan o'tishni yoqish/o'chirish, Google OAuth yoqilganmi, maintenance mode va xabari, default learner theme/accent, ruxsat etilgan upload hajmi, learner wallpaper ruxsati); saqlash modeli (typed jadval vs key-value + JSON schema validatsiya); qaysi RBAC rollari o'zgartira oladi; maintenance mode'da owner kirishi va learner'ga ko'rinish; secretlar (API kalitlar) bu yerda saqlanmaydi — faqat `.env`; sozlamalarni keshlash va boshqa modullarga yetkazish (public read endpoint vs server-side config).

## 4.07-Owner · Audit Logs — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 4.01
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §28, §21 (21.1, 21.3), §6.2 o'qildi.
Testlar: —
Muammolar: Audit Logs sahifasi (owner harakatlari ro'yxati: vaqt, actor, action, target turi/ID, IP/user agent, o'zgarish tafsilotlari; filterlar — actor, action, target turi, sana oralig'i; qidiruv; pagination; yozuv tafsiloti) faqat o'qish uchun va quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: `audit_logs` jadvali va uning sxemasi (ustunlar, metadata JSONB). Filter/sort uchun qo'shimcha indekslarni 4.07 o'z migrationi bilan qo'shadi.
- 0.02: UI kit (table, filter, date range picker, badge, drawer/modal, pagination, skeleton, empty/error state).
- 1.02: owner RBAC middleware (audit log'ni kim ko'ra oladi) va audit yozish helperi — yozuvlar formati shu helperdan keladi. 4.07 yozuv yaratmaydi, faqat o'qiydi.
- 4.01: owner layout va sidebar (Platform → Audit Logs havolasi).
Yozuvlar mazmuni boshqa owner modullari (3.xx CMS, 4.03, 4.06, 3.13 va h.k.) helper orqali yozgani sari to'ladi; yozuv bo'lmasa empty state ko'rsatiladi, demo yozuvlar ko'rsatilmaydi.
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.02 → 4.01 tugagach 4.07 ni boshlash (tavsiya etilgan tartibda 4.06 dan keyin). Avval reja beriladi va tasdiq olinadi: audit log'ni ko'rish huquqi (faqat `owner` roli yoki boshqa rollar ham — 1.02 rollariga mos); yozuvlar immutable ekanligi (UPDATE/DELETE taqiqlanadi, DB darajasida trigger yoki faqat app darajasida); pagination turi (cursor vs offset — katta jadval uchun cursor tavsiya etiladi); metadata'dagi sezgir maydonlarni (password, token, secret) maskalash; retention siyosati (qancha saqlanadi, arxivlash); CSV eksport kerakmi va eksportning o'zi audit logga yoziladimi; audit log'ni ko'rish ham audit logga yoziladimi; 4.02 dagi "so'nggi admin harakatlari" bloki bilan umumiy endpoint.

## 4.05-Owner · AI Content Assistant — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 3.01, 3.02, 3.12, 4.01 (draft turlari uchun qisman 3.03, 3.05, 3.10)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §23 (23.1–23.3), §22 va PROGRESS'dagi 3.01, 3.03, 3.12 yozuvlari o'qildi.
Testlar: —
Muammolar: AI Content Assistant ("Create with AI" flow: domain → topic → kontent turi → talablar → AI structured draft → editorda ochiladi → owner tahrirlaydi → preview → Save Draft/Publish; AI hech qachon avtomatik publish qilmaydi; imkoniyatlar: lesson/savol explanation, worked example, practice savol taklifi, qo'shimcha misollar, soddalashtirish, grammatik/matematik tekshiruv, tarjima, domain/topic mosligini tekshirish) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware, rate limiter), `api/internal/<module>/` va `web/` tuzilmasi. 0.01: migration tool, `users`, `audit_logs`. 0.02: UI kit (form, select, stepper, modal, preview bloki, validation xabarlari).
- 1.02: owner auth, RBAC middleware, audit helper (AI draft yaratish, qabul qilish/rad etish audit logga yoziladi). 4.01: owner layout va sidebar (Content Management → AI Content Assistant).
- 3.12: AI provider client, structured (JSON schema) javob va validatsiya, mathematical validation hook, rate limit, usage/cost hisobi, `ai_requests` loglari. 4.05 provider'ni to'g'ridan-to'g'ri chaqirmaydi, kalitlar frontendga chiqmaydi — faqat 3.12 interfeysi orqali.
- 3.01: Draft/In Review/Published/Archived workflow va CMS skeleti — AI natijasi faqat Draft sifatida shu workflow'ga tushadi. 3.02: domain/topic ID'lari (flow'ning 1–2 qadamlari va moslik tekshiruvi).
- Qisman (draft maqsadi): 3.03 lesson editor/API (lesson, explanation, worked example), 3.05 question bank va versiyalari (practice savollar: to'g'ri javob va explanation mosligi tekshiriladi), 3.10 formula/concept yozuvlari. 4.05 bu editorlarni yasamaydi — draftni ularning create-draft API'si orqali yaratib, o'sha editorda ochadi. Tayyor bo'lmagan kontent turi ro'yxatda ko'rsatilmaydi.
Keyingi qadam: tavsiya etilgan tartibda 2.06 dan keyin (3.12 tugagach) 4.05 ni boshlash. Avval reja beriladi va tasdiq olinadi: AI draftlar va statusi qayerda saqlanadi (alohida `ai_content_drafts` jadvali — prompt, javob JSON, validatsiya natijasi, bog'langan kontent ID — vs faqat 3.01 draftlari + `ai_requests`); har bir kontent turi uchun JSON schema; matematik validatsiya usuli (deterministik checker vs ikkinchi AI tekshiruvi — 3.12 bilan kelishiladi); validatsiyadan o'tmagan draft editorda ogohlantirish bilan ochiladimi yoki bloklanadimi; soddalashtirish/tarjima mavjud kontentga diff ko'rinishida qo'llanadimi; owner uchun kunlik limit va cost cap; tushunarli xato xabarlari; unit (fake provider bilan), integratsiya va komponent testlari.

## 4.04-Owner · Analytics — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 4.01 (metrikalar uchun qisman 1.01, 3.02, 3.03/3.04, 3.06, 3.07/3.08, 3.10, 3.13)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §26 (va kontekst uchun §21.2, §25) o'qildi.
Testlar: —
Muammolar: Analytics sahifasi 4.01 owner layouti ichida ishlaydi va boshqa modullar ma'lumotini faqat o'qib aggregatsiya qiladi (o'zi yasamaydi). Asosiy bog'liqliklar:
- 0.00: Go core (config, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi. 0.01: migration tool, `users`, `profiles`, `audit_logs`. 0.02: UI kit (stat card, chart wrapper, date range picker, table, skeleton, empty/error state). 1.02: owner session va RBAC middleware (endpointlar faqat owner uchun). 4.01: owner layout va `/owner/analytics` route.
- Metrika manbalari (§26): registrations over time — `users` (0.01); active learners — learner sessionlari/faollik (1.01) va practice/lesson/exam hodisalari; practice completion — 3.06; mock exam completion — 3.07/3.08; eng ko'p o'rganiladigan domainlar — 3.02 taksonomiyasi + 3.04 lesson progress va 3.06 practice; kontentdan foydalanish — 3.04 (lessons), 3.06 (questions), 3.10 (formulas); subscription metrics — 3.13.
Manba modul hali qurilmagan metrika uchun empty state ("ma'lumot hali mavjud emas") ko'rsatiladi. Hammasi haqiqiy ma'lumotdan hisoblanadi, demo yoki fake raqam ko'rsatilmaydi (§26).
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.02 → 4.01 va asosiy manba modullar tugagach 4.04 ni boshlash (tavsiya etilgan tartibda 4.03 dan keyin). Avval reja beriladi va tasdiq olinadi: metric definitionlar (4.02 bilan bir xil bo'lishi kerak — "active learner" ta'rifi, "completion" nima, tasdiqlanmagan emaillar hisoblanadimi) va ularni `docs/` da bitta joyda hujjatlash; vaqt oraliqlari (7/30/90 kun, custom) va granularity (kun/hafta/oy), timezone; hisoblash usuli (har so'rovda SQL aggregatsiya vs kunlik rollup jadvali/materialized view + keshlash); umumiy activity/event jadvali kerakmi yoki har modul jadvalidan o'qiladimi; chart kutubxonasi (yangi dependency qo'shiladimi yoki 0.02 komponentlari); CSV eksport; analytics ko'rish audit logga yoziladimi; learner shaxsiy ma'lumotlarini oshkor qilmaslik (faqat aggregatlar).

## 4.03-Owner · Learners — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 4.01 (learner profil bloklari uchun qisman 2.01, 2.07, 3.06, 3.08/3.09, 3.13)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §25, §21 (21.1, 21.3), §28 o'qildi.
Testlar: —
Muammolar: Learner Management (learnerlar jadvali: avatar, name, email, registration date, account status, subscription plan, last activity — qidiruv, filter, sort, pagination; learner profili: account ma'lumotlari, learning progress, practice activity, mock exam tarixi, subscription holati; owner password/hash'ni hech qachon ko'rmaydi; admin harakatlari audit logga yoziladi) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: `users`, `profiles`, `audit_logs` jadvallari (account status maydoni, last activity manbai). Ro'yxat uchun kerakli indekslarni 4.03 o'z migrationi bilan qo'shadi.
- 0.02: UI kit (table, avatar, badge, filter, pagination, empty/error state).
- 1.02: owner RBAC middleware va audit yozish helperi — barcha `/owner/learners` endpointlari shu orqali himoyalanadi.
- 4.01: owner layout va sidebar (Overview → Learners havolasi).
- Profil bloklari uchun ma'lumot manbalari (4.03 ularning logikasini yasamaydi, faqat o'qiydi): 2.01 (avatar, profil), 2.07 (learning progress), 3.06 (practice activity), 3.08/3.09 (mock exam tarixi va natijalari), 3.13 (subscription plan va holati). Ular tayyor bo'lmasa, tegishli blok "mavjud emas" holatida qoladi — fake data ko'rsatilmaydi.
Keyingi qadam: Tavsiya etilgan tartib bo'yicha 0.00 dan boshlash kerak. 0.00, 0.01, 0.02, 1.02 va 4.01 tayyor bo'lgach, 4.03 shu yozuvdan davom ettiriladi. Avval reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi: owner learner account statusini o'zgartira oladimi (suspend/reactivate) va qaysi RBAC rollari uchun; learner'ni o'chirish faqat learnerning o'zi (2.10) orqalimi yoki owner ham qila oladimi; "last activity" manbai (session/request vaqti vs alohida activity yozuvi); qidiruv usuli (ILIKE vs full-text); learner profilini ko'rish ham audit logga yoziladimi (maxfiylik); CSV eksport kerakmi.

## 4.02-Owner · Dashboard Home — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 4.01 (metrikalar uchun qisman 1.01, 3.01, 3.03, 3.05, 3.06, 3.07/3.08, 3.13)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §21 (ayniqsa §21.2) o'qildi.
Testlar: —
Muammolar: Owner Dashboard Home 4.01 owner layouti ichida ishlaydi va boshqa modullar ma'lumotini faqat o'qib yig'adi (o'zi yasamaydi). Asosiy bog'liqliklar:
- 0.00: Go core (config, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi. 0.01: migration tool, `users`, `audit_logs`. 0.02: UI kit (stat card, table, skeleton, empty/error state). 1.02: owner session, RBAC middleware (endpoint faqat owner uchun). 4.01: owner layout va `/owner` route.
- Metrika manbalari: total learners / new registrations / active learners — `users` (0.01) va learner sessionlari yoki faollik (1.01); practice activity — 3.06; completed mock exams — 3.07/3.08; content inventory, published va draft — 3.01 workflow statuslari (3.03 lessons, 3.05 questions, 3.07 exams, 3.10 formulas); subscription stats — 3.13; so'nggi admin harakatlari — `audit_logs` (0.01, helper 1.02).
Manba modul hali qurilmagan blok uchun empty state ("ma'lumot hali mavjud emas") ko'rsatiladi. Demo yoki fake raqam production statistikasi sifatida ko'rsatilmaydi (§21.2).
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.02 → 4.01 tugagach 4.02 ni boshlash (tavsiya etilgan tartibda 3.14 dan keyin). Avval reja beriladi va tasdiq olinadi: bitta aggregatsiya endpointi (`GET /owner/dashboard`) vs blok bo'yicha endpointlar; "active learner" ta'rifi (oxirgi 7/30 kunda login yoki practice/lesson faolligi); davr tanlovi (bugun/7 kun/30 kun) va timezone; "new registrations" — tasdiqlangan email bo'yichami; keshlash (masalan 1–5 daqiqa) yoki har so'rovda hisoblash; so'nggi admin harakatlari soni va 4.07 Audit Logs sahifasiga havola.

## 4.01-Owner · Dashboard UI — 2026-10-11
Status: Blocked by 0.00, 0.02, 1.02
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `docs/ARCHITECTURE.md` hali mavjud emas. PRD §21 (21.1, 21.3) o'qildi.
Testlar: —
Muammolar: Owner layout (sidebar: Overview — Dashboard, Learners, Analytics; Content Management — Curriculum CMS, Lessons, Question Bank, Mock Exam Builder, Formula Library, AI Content Assistant; Platform — Plans & Subscriptions, Notifications, Platform Settings, Audit Logs; header; mobile drawer; owner theme — Light/Dark/System, sidebar holati, compact layout, ruxsat berilsa accent; learner theme'dan mustaqil) quyidagilarga tayanadi:
- 0.00: `web/` (Next.js App Router) tuzilmasi, `/owner` route group joylashuvi.
- 0.02: design tokenlar, theme mexanizmi va UI kit (Button, IconButton, Dropdown, Drawer, Avatar, Tooltip). Owner theme shu token tizimi ustiga quriladi, alohida rang tizimi yasalmaydi.
- 1.02: owner session va route guard (layout faqat login qilgan owner uchun, RBAC rolga qarab sidebar bandlarini ko'rsatish), logout endpointi, header uchun joriy owner ma'lumoti.
Eslatma: sidebar'dagi sahifalar (4.02–4.07, 3.xx CMS'lar) 4.01 da yasalmaydi — faqat havolalar. Header'dagi notifications (3.14) ma'lumoti faqat slot sifatida qoldiriladi.
Keyingi qadam: 0.00 → 0.02 → 1.02 tugagach (tavsiya etilgan tartibda 2.03 dan keyin), 4.01 ni boshlash. Avval reja beriladi va tasdiq olinadi: owner theme sozlamalari qayerda saqlanadi (localStorage vs alohida owner preferences — learner `user_preferences` dan mustaqil bo'lishi kerak), compact layout nimani o'zgartiradi (spacing/qator balandligi), accent owner uchun ruxsat etiladimi, hali qurilmagan sahifalar havolalari (yashirish vs "tez orada"), 2.03 learner layout bilan umumiy shell komponentlari qayta ishlatiladimi.

## 3.14-Learner & Owner · Notifications — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 1.02, 2.03, 4.01 (bildirishnoma manbalari uchun qisman 2.05, 2.07, 2.11, 3.13)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §31, §20 (Notifications), §10.3, §21 (Owner Platform → Notifications), §3 (Notifications entity) o'qildi.
Testlar: —
Muammolar: Notifications (in-app bildirishnomalar: study plan reminder, tugallanmagan vazifa, haftalik summary, security event, subscription holati; email kanali; foydalanuvchi sozlamalariga mos yuborish; Owner notifications) quyidagilarga tayanadi:
- 0.00: monorepo, Go core (config, logger, error handler, middleware), `ARCHITECTURE.md`, email yuborish va job runner uchun joy. 0.01: migration tool, `users` va `user_preferences` jadvallari (`notifications` jadvalini shu sessiya migration orqali qo'shadi).
- 0.02: UI kit (dropdown, badge, toast, empty state). 2.03: learner header'dagi notifications slot. 4.01: owner layout (Owner → Platform → Notifications sahifasi).
- 1.01: learner auth va sessionlar (bildirishnoma egasi, security event manbai). 1.02: RBAC middleware va audit helper (Owner notifications endpointlari).
- Manbalar (event yuboruvchi modullar): 2.05 study plan va vazifalar (reminder, tugallanmagan vazifa), 2.07 haftalik summary ma'lumoti, 3.13 subscription holati. 3.14 bu modullar uchun emit API (interfeys) beradi, ularning logikasini o'zi yasamaydi.
- 2.11: notification toggle UI va preference saqlash. 3.14 kanal va tur modelini hamda yuborishda preference tekshiruvini qiladi.
- Eslatma: 2.11 va 2.03 yozuvlaridagi "Notifications uchun 3.13" eski raqamlash. Joriy xaritada Notifications = 3.14, Subscription = 3.13.
Keyingi qadam: Tavsiya etilgan tartib bo'yicha 0.00 dan boshlash kerak. 0.00, 0.01, 0.02, 1.01, 1.02, 2.03 va 4.01 tayyor bo'lgach, 3.14 shu yozuvdan davom ettiriladi.

## 3.12-Learner & Owner · AI Core — 2026-10-11
Status: Blocked by 0.00, 0.01, 1.01, 1.02 (Owner AI usage sahifasi uchun 0.02, 4.01; plan bo'yicha limitlar uchun qisman 3.13)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §3, §19, §23 (ayniqsa §23.3), §27 va PROGRESS'dagi 2.06 yozuvi o'qildi.
Testlar: —
Muammolar: AI Core (provider client interfeysi, API kalitlari faqat `.env`da va frontendga chiqmaydi, strukturali (JSON schema) javob olish va validatsiya, matematik validation hook'i, per-user/per-feature usage limitlar, rate limit, token/cost hisobi va budget cap, `ai_requests` loglari, AI kontenti va statusini saqlash, Owner uchun usage/cost ko'rinishi) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware, rate limiter), `api/internal/<module>/` tuzilmasi. 3.12 0.00 dagi rate limiter ustiga quriladi, o'zinikini yozmaydi.
- 0.01: migration tool, `users`, `audit_logs`. `ai_requests` (usage/cost log), `ai_usage_counters` va kerak bo'lsa `ai_limits` jadvallarini 3.12 o'z migrationi bilan qo'shadi. `ai_conversations`/`ai_messages` 2.06 ga tegishli.
- 1.01: learner auth middleware (learner AI endpointlari uchun user identifikatsiyasi). 1.02: owner RBAC va audit helper (limit/budget sozlamalarini o'zgartirish audit logga yoziladi).
- 0.02, 4.01: Owner tomonidagi AI usage/cost sahifasi uchun UI kit va owner layout.
- 3.13 (qisman): Free/Premium plan bo'yicha limitlar. 3.13 tayyor bo'lmaguncha limitlar config'dan (default Free) olinadi; plan ulanishi uchun interfeys qoldiriladi.
Eslatma: 2.06 yozuvi eski raqamlashdan foydalanadi ("3.11: AI Core", "3.12: plan limitlari"). Joriy xaritada AI Core = 3.12, Subscription & Plans = 3.13. 2.06 va 4.05 AI provider'ni to'g'ridan-to'g'ri chaqirmaydi, faqat 3.12 interfeysi orqali ishlaydi.
Keyingi qadam: 0.00 → 0.01 → 1.01 → 1.02 (Owner sahifasi uchun 0.02, 4.01) tugagach 3.12 ni boshlash. Avval reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi: AI provider va model (tavsiya: Anthropic Claude, provider-agnostik interfeys ortida; yangi dependency — rasmiy Go SDK yoki oddiy `net/http` — tasdiq kerak); Free va Premium uchun default kunlik/oylik limitlar va global oylik budget cap; streaming (SSE) Core darajasida qo'llab-quvvatlanadimi; promt/javob matnlari logda qancha muddat saqlanadi (maxfiylik); matematik validation usuli (AI javobini qayta hisoblash/ikkinchi tekshiruv vs deterministik checker); limitga yetilganda learnerga ko'rsatiladigan xabar; unit (fake provider bilan), integratsiya va rate-limit testlari.

## 3.13-Learner & Owner · Subscription & Plans — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 1.02, 2.03, 4.01 (usage limitlar uchun qisman 3.12)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §27 va §20 (Subscription) o'qildi.
Testlar: —
Muammolar: Subscription & Plans (Free/Premium plan definitions, backendda feature gating va usage limitlar, subscription status, billing cycle, payment history, cancellation, renewal, payment failure handling; Learner Settings → Subscription tab; Owner subscriptions sahifasi) quyidagilarga tayanadi:
- 0.00: Go core (config, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi. 0.01: migration tool, `users`, `audit_logs`. 0.02: UI kit (plan card, badge, table, modal).
- 1.01: learner auth (subscription egasi). 1.02: owner RBAC va audit helper (plan o'zgartirish, qo'lda premium berish/bekor qilish audit logga yoziladi).
- 2.03 / 4.01: learner va owner layoutlari (sidebar havolalari).
- 3.12 (qisman): AI usage limitlari plan'ga bog'lanadi — 3.13 `entitlements` servisini beradi, 3.12 uni chaqiradi.
- Iste'molchilar: 2.10/2.11 (Settings'dagi Subscription tab — eslatma: 2.10 yozuvidagi "Subscription tab uchun 3.12" eski raqamlash, joriy xaritada = 3.13), 3.06/3.07/3.08 (premium kontent gating), 4.02/4.04 (MRR, konversiya metrikalari), 5.01 (haqiqiy provayder va webhooklar).
Keyingi qadam: bloklar tugagach (tavsiya etilgan tartibda 3.11 → 2.05 → 3.12 → ... → 2.11 dan keyin) reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi: Free va Premium farqlari (qaysi feature'lar va qanday limitlar: mock exam soni, AI so'rovlar, practice); billing cycle (oylik/yillik) va narxlar Owner tomonidan sozlanadimi; payment provider abstraksiyasi (`PaymentProvider` interfeysi + 5.01 gacha faqat Owner qo'lda beradigan "manual" provayder, fake to'lov emas); cancel = davr oxirigacha access saqlanadimi; payment failure uchun grace period muddati; trial bo'ladimi.

## 3.11-Learner & Owner · Placement Test — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 1.02, 2.03, 3.01, 3.02, 3.05, 4.01 (oqim uchun qisman 2.02)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §9, §9.1, §8.3 o'qildi.
Testlar: —
Muammolar: Placement Test (Math va R&W diagnostik savollari, moslashuvchan savollar banki va scoring modeli, diagnostik profil: kuchli/zaif domainlar, tavsiya etilgan boshlang'ich mavzular, study plan uchun boshlang'ich ma'lumot; natija rasmiy SAT score emas) quyidagilarga tayanadi:
- 0.00: monorepo, Go core, `ARCHITECTURE.md`. 0.01: migration tool, users jadvali. 0.02: UI kit va tokenlar.
- 1.01: learner auth (attempt egasi). 1.02: RBAC va audit helper (Owner placement konfiguratsiyasi). 2.03 / 4.01: learner va owner layoutlari.
- 3.01: Draft/In Review/Published workflow (placement test konfiguratsiyasini publish qilish).
- 3.02: domain/topic ID'lari (domain breakdown va tavsiya mavzular). 3.05: savollar va versiyalari, MCQ/numeric render va tekshirish — placement savollari shu bankdan olinadi, alohida bank yaratilmaydi.
- 2.02: onboarding oxiri Placement Assessment sahifasiga olib boradi (PRD §8.3) va §9 dagi uch yo'l tanlovi (oldingi natija / placement test / testsiz).
- Iste'molchilar: 2.05 (study plan kirishi), 2.04, 2.07, 2.06 diagnostik profilga tayanadi. Eslatma: 2.06 yozuvidagi "3.11: AI Core" eski raqamlash; joriy xaritada AI Core = 3.12.
Keyingi qadam: bloklar tugagach (tavsiya etilgan tartibda 3.09 dan keyin) reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi: test adaptiv (Digital SAT kabi 2 bosqichli modul) yoki statik bo'ladimi; savollar soni va vaqt limiti; scoring modeli (domain bo'yicha to'g'ri foiz + og'irliklar, konfiguratsiya Owner tomonidan versiyalanadi) va taxminiy oraliq ko'rsatiladimi; qayta topshirish qoidasi; "kuchli/zaif" chegaralari; §9 dagi 1-yo'l (oldingi SAT natijasi) diagnostik profilni qanday to'ldiradi.

## 3.10-Learner & Owner · Formula & Concept Library — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 1.02, 2.03, 3.01, 3.02, 4.01 (related practice questions uchun qisman 3.05)
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §18 va PROGRESS'dagi 3.01, 3.02, 3.05 yozuvlari o'qildi.
Testlar: —
Muammolar: Formula & Concept Library (har bir yozuvda: nom, formula, izoh, qayerda ishlatilishi, misol, tegishli mavzu, related practice questions; R&W uchun grammar va vocabulary tushunchalari; search va bookmark; Owner CMS va Learner kutubxonasi) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware, rate limiter), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool, `users`/`audit_logs`. `library_entries` (formula | concept), `library_entry_questions`, `library_bookmarks` jadvallarini 3.10 o'z migrationi bilan qo'shadi.
- 0.02: design tokenlar va UI kit (search input, filter chips, card, tabs, bookmark tugmasi, formula render bloki, empty state).
- 1.01: learner auth middleware (kutubxona va bookmark endpointlari). 1.02: owner RBAC va audit helper (create/update/publish/archive audit logga yoziladi).
- 3.01: Draft/In Review/Published/Archived workflow va CMS skeleti — Library bo'limi shu skeletga ulanadi; Learner faqat Published yozuvlarni ko'radi.
- 3.02: section/domain/topic taksonomiyasi ("tegishli mavzu" FK va filterlar).
- 3.05 (qisman): "related practice questions" faqat published `questions` ID'lariga bog'lanadi; 3.05 tayyor bo'lmasa bu maydonsiz boshlab, keyin ulash mumkin.
- 2.03 / 4.01: learner va owner layoutlari (sidebar havolalari).
Keyingi qadam: 0.00 → 0.01 → 0.02 → 1.01 → 1.02 → 2.03 → 4.01 → 3.01 → 3.02 (va imkon bo'lsa 3.05) tugagach 3.10 ni boshlash. Avval reja beriladi va tasdiq olinadi: formula formati (LaTeX + KaTeX render — yangi dependency, tasdiq kerak — vs oddiy matn); bitta `library_entries` jadvali `kind` bilan vs formula/concept uchun alohida jadvallar; qidiruv usuli (PostgreSQL full-text `tsvector` + trigram vs oddiy `ILIKE`); bookmark 2.08 Saved Questions bilan umumiy modelmi yoki alohida; versiyalash kerakmi (published yozuv tahriri yangi draft); route'lar (`/library`, `/library/<slug>`, owner CMS ichida Library bo'limi); loading/empty/error statelar; unit, integratsiya va komponent testlari.

## 3.09-Learner · Exam Results & Review — 2026-10-11
Status: Blocked by 3.07 (u esa 0.00, 0.01, 0.02, 1.02, 3.01, 3.02, 3.05, 4.01 ga bloklangan) va 3.08; shuningdek 1.01, 2.03
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §15, §14 va PROGRESS'dagi 3.07/3.08 yozuvlari o'qildi.
Testlar: —
Muammolar: Exam Results & Review (umumiy performance, Math va R&W natijalari, to'g'ri/noto'g'ri/javobsiz savollar, domain bo'yicha natijalar, vaqt sarfi, savolma-savol review, to'g'ri javoblar va explanationlar, tavsiya mavzular, Practice Again; scoring validatsiya qilinmaguncha rasmiy score emas, practice performance yoki estimated score range) quyidagilarga tayanadi:
- 3.07: attempt/answer jadvallari, scoring va Learner natija/review API (attempt natijasi, domain breakdown, savolma-savol review — answer key va explanation faqat attempt yakunlangach, faqat attempt egasiga). Juft sessiya qoidasiga ko'ra 3.09 backend/DB/scoring yozmaydi, faqat `docs/API.md` dagi 3.07 kontraktidan foydalanadi; API o'zgarishi kerak bo'lsa 3.07 ga yo'naltiriladi.
- 3.08: submitdan keyin natija sahifasiga o'tish (route va attempt ID kelishiladi).
- 0.00: `web/` tuzilmasi. 0.02: UI kit va tokenlar (stat kartalar, breakdown jadvali/grafik, savol review kartasi, formula rendering). 1.01: learner auth. 2.03: learner layout.
- 3.02: domain/topic ID'lari (tavsiya mavzular havolalari). 3.05: savol kontenti renderi (MCQ/numeric) va explanation formati.
- Qisman: 3.06 (Practice Again — zaif domainlar bo'yicha practice set ochish), 2.08 (review'dagi bookmark tugmasi), 2.09 (noto'g'ri javoblar Mistake Notebook'ga tushishi). Tayyor bo'lmasa tugmalar ko'rsatilmaydi, fake ma'lumot yo'q.
- Iste'molchilar: 2.04, 2.05, 2.07 exam natijalariga tayanadi.
Keyingi qadam: 3.07 natija/review kontraktini `docs/API.md` ga yozgach va 3.08 tugagach, 3.09 ni boshlash. Avval reja beriladi: route'lar (`/exams/attempts/<id>/results` va `/review`, learner layout'da, exam layout'da emas); score ko'rsatish qoidasi (3.07 scoring holatiga qarab practice performance % yoki estimated range + "rasmiy SAT score emas" disclaimer); review filterlari (hammasi/noto'g'ri/javobsiz/marked), modul bo'yicha navigatsiya; tugallanmagan yoki boshqa userning attemptiga kirish holatlari; loading/empty/error statelar; unit va komponent testlari.

## 3.08-Learner · Mock Exam Runner — 2026-10-11
Status: Blocked by 3.07 (u esa 0.00, 0.01, 0.02, 1.02, 3.01, 3.02, 3.05, 4.01 ga bloklangan); shuningdek 1.01, 2.03
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §14 va PROGRESS'dagi 3.07 yozuvi o'qildi.
Testlar: —
Muammolar: Mock Exam Runner (exam katalogi, instructions, server-side timer, autosave, recovery, idempotent submit) quyidagilarga tayanadi:
- 3.07: exam/attempt/answer jadvallari va Learner exam API (katalog, start, modul bo'yicha `deadline_at`, autosave, modulni yakunlash, idempotent submit, tanaffus). Juft sessiya qoidasiga ko'ra 3.08 backend/DB yozmaydi, faqat `docs/API.md` dagi 3.07 kontraktidan foydalanadi; API o'zgarishi kerak bo'lsa 3.07 ga yo'naltiriladi.
- 0.00: `web/` tuzilmasi. 0.02: UI kit va tokenlar (savol karta, variantlar, timer, navigator, modal).
- 1.01: learner auth. 2.03: exam uchun bo'sh (sidebar'siz) layout.
- 3.05: savol renderi formati (MCQ/numeric, formula/rasm) — runner savollarni answer keysiz ko'rsatadi.
- Iste'molchi: 3.09 (submitdan keyin natija sahifasiga o'tish).
Keyingi qadam: 3.07 tugab `docs/API.md` ga Learner exam kontraktini yozgach, 3.08 ni boshlash. Avval reja beriladi: route'lar (`/exams` katalog, `/exams/<id>` instructions, `/exams/attempts/<id>` runner, exam layout'da); timer faqat server `deadline_at` va server vaqti offsetidan hisoblanadi (client faqat ko'rsatadi, timeoutda server yakunlaydi); autosave (debounce + muntazam flush, offline navbat va qayta urinish, "Saved/Saving/Offline" indikatori); refresh/tarmoq uzilishidan keyin attempt holatini serverdan tiklash; Mark for Review, navigator, answered indikatori, Next/Previous, modulni yakunlash tasdig'i, tanaffus ekrani; submit idempotency key bilan, ikki marta bosilishdan himoya; loading/empty/error statelar; unit, komponent va (5.04 bilan) E2E testlar.

## 3.07-Owner · Mock Exam Builder — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 3.01, 3.02, 3.05, 4.01
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §14 (14.1–14.2, kontrakt uchun 14.5–14.6 ham), §22.1 va §24 o'qildi.
Testlar: —
Muammolar: Mock Exam Builder (exam backend, API, DB va Owner builder: title/description, exam type — Full-Length, Math, R&W, Section-specific; module configuration; question selection; domain va difficulty distribution; time configuration; preview; publish oldidan validatsiya; Publish/Unpublish/Archive; published exam versiyalari) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool, `users`/`audit_logs`. `mock_exams`, `mock_exam_versions`, `mock_exam_modules`, `mock_exam_module_questions` jadvallarini 3.07 o'z migrationi bilan qo'shadi.
- 0.02: design tokenlar va UI kit (form, tabs/stepper, table, tartiblanadigan ro'yxat, badge, modal, validation xabarlari).
- 1.02: owner auth, RBAC middleware, audit helper (create/update/publish/unpublish/archive/new version audit logga yoziladi).
- 3.01: Draft/In Review/Published/Archived workflow va CMS skeleti. 3.07 o'z workflow'ini yasamaydi, Mock Exams bo'limini shu skeletga ulaydi.
- 3.02: section/domain taksonomiyasi (domain distribution va validatsiya).
- 3.05: `questions`/`question_versions` (exam versiyasi aniq `question_version_id` larga bog'lanadi), difficulty, filterli savol qidirish API'si. Bularsiz question selection yasab bo'lmaydi.
- 4.01: owner layout va sidebar.
- Juft sessiya qoidasi: attempt/answer jadvallari va Learner exam API (katalog, start, server-side deadline, autosave, idempotent submit — 3.08 uchun; natija/review — 3.09 uchun) ham 3.07 tomonidan quriladi va `docs/API.md` ga yoziladi. 3.08/3.09 faqat shu API'dan foydalanadi.
Keyingi qadam: 0.00 → 0.01 → 0.02 → 1.01 → 1.02 → 2.03 → 4.01 → 3.01 → 3.02 → 3.05 tugagach (tavsiya etilgan tartibda 3.10 dan keyin) 3.07 ni boshlash. Avval reja beriladi va tasdiq olinadi: ma'lumot modeli (`mock_exams` stabil ID + immutable `mock_exam_versions`; publish'da versiya muzlatiladi, attemptlar `exam_version_id` ga bog'lanadi, published examni tahrirlash yangi draft versiya yaratadi); rasmiy format presetlari (R&W 2×27 savol, 2×32 daq; Math 2×22 savol, 2×35 daq; 10 daq tanaffus) Full-Length/Math/R&W uchun qulflanadimi, vaqt faqat Section-specific/practice formatlarda erkinmi (aniq ko'rsatiladi); savol tanlash usuli (qo'lda vs distribution bo'yicha avto-to'ldirish + qo'lda almashtirish; V1 fixed-form, adaptive routing yo'q — §14.6); validatsiya qoidalari (modul savollar soni, takroriy savol yo'q, faqat Published savol versiyalari, section mosligi, domain/difficulty distribution chegaralari); bitta savol bir nechta examda ishlatilishi mumkinmi; Unpublish faol attemptlarga ta'sir qilmasligi; 3.08 uchun attempt modeli (modul bo'yicha server-side `deadline_at`, autosave, idempotency key, ikki marta yakunlanmaslik) va 3.09 uchun scoring (practice performance yoki taxminiy oraliq, rasmiy score emas).

## 3.06-Learner & Owner · Daily Practice — 2026-10-11
Status: Blocked by 3.05 (u esa 0.00, 0.01, 0.02, 1.02, 3.01, 3.02, 4.01 ga bloklangan); shuningdek 1.01, 2.03
Bajarildi: Hech narsa. Repo tekshirildi: faqat `CLAUDE.md` va `docs/` (`PRD.md`, `PROGRESS.md`) bor; `api/`, `web/`, `api/migrations/`, `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §12.4, §12.5, §13 o'qildi.
Testlar: —
Muammolar: Daily Practice (practice setlar, Practice va Test mode, javobni tekshirish, explanation, set oxiridagi natija; Owner tomonda practice setlarni yaratish/publish qilish) quyidagilarga tayanadi:
- 0.00: Go core, `api/internal/<module>/` va `web/` tuzilmasi. 0.01: migration tool. 0.02: UI kit (savol karta, variantlar, progress bar, natija ekrani).
- 1.01: learner auth. 1.02: owner RBAC va audit helper (practice set create/publish/archive). 2.03 / 4.01: learner va owner layoutlari.
- 3.01: workflow va CMS skeleti (practice setlar bo'limi shunga ulanadi). 3.02: topic katalogi (12.4 qadam 1 — learner topic tanlaydi).
- 3.05: `questions` / `question_versions` va **javob tekshirish funksiyasi** (MCQ + numeric). 3.06 o'z tekshiruv logikasini yasamaydi, 3.05 dagini qayta ishlatadi; attemptlar `question_version_id` ga bog'lanadi.
- Qisman: 2.08 (12.4 qadam 7 — savolni saqlash; bookmark tugmasi 2.08 API'siga ulanadi), 2.09 (noto'g'ri javobdan keyin Mistake Notebook hook'i).
- Iste'molchilar: 2.04, 2.05, 2.07, 3.04 (lesson ichidagi practice), 3.11 practice attempt ma'lumotlariga tayanadi.
Keyingi qadam: 3.05 tugab `docs/API.md` ga savol va answer-checking kontraktini yozgach, 3.06 ni boshlash. Avval reja beriladi va tasdiq olinadi: ma'lumot modeli — `practice_sets` (Owner tomonidan tuzilgan, topic'ga bog'langan, status workflow 3.01 dan) vs topic bo'yicha dinamik yig'iladigan setlar (yoki ikkalasi); `practice_attempts` (user, set, mode: practice|test, started_at, completed_at, score) va `practice_answers` (attempt, question_version_id, javob, is_correct, time_spent_ms; unique (attempt, question)); Practice mode'da har bir javob darhol serverda tekshirilib explanation qaytariladi, Test mode'da answer key va explanation faqat attempt yakunlangach beriladi (backendda majburiy); yakunlash idempotent; tugallanmagan attemptni davom ettirish; "Daily" ma'nosi (kunlik tavsiya etilgan set 2.05 reja bilan kelishiladi); loading/empty/error statelar; unit, integratsiya va komponent testlari.

## 3.05-Owner · Question Bank — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 3.01, 3.02, 4.01
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §12.5, §13 va §22 (22.1 status workflow) o'qildi.
Testlar: —
Muammolar: Question Bank (savollar va versiyalari, MCQ va numeric turlar, filterlar: backend, API, DB va Owner CMS) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware, rate limiter), storage interfeysi (savol rasmlari/grafiklari uchun), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool va `users`/`audit_logs` jadvallari. `questions`, `question_versions`, `question_choices` (yoki versiya ichidagi JSONB), `question_numeric_answers` jadvallarini 3.05 o'z migrationi bilan qo'shadi.
- 0.02: design tokenlar va UI kit (table, filter bar, form, select, modal, badge, preview).
- 1.02: owner auth, RBAC middleware va audit helper (create/update/publish/archive/new version audit logga yoziladi).
- 3.01: Draft/In Review/Published/Archived workflow, CMS skeleti va preview. 3.05 o'z workflow'ini yasamaydi, Question Bank bo'limini 3.01 skeletiga ulaydi.
- 3.02: section/domain/topic/subtopic taksonomiyasi (savol → topic FK, section va domain filterlari).
- 4.01: owner layout va sidebar.
Keyingi qadam: yuqoridagi sessiyalar tugagach 3.05 ni boshlash. Avval reja beriladi: ma'lumot modeli — `questions` (stable ID, section, domain/topic, type, status, current_version_id, source, license) + immutable `question_versions` (content, choices, answer key, explanation, difficulty, version raqami; published versiya tahrirda o'zgarmaydi, yangi versiya yaratiladi, attemptlar `question_version_id` ga bog'lanadi); `type` enum + type'ga xos answer schema (mcq: choices + correct choice; numeric: qabul qilinadigan qiymatlar/kasr/ekvivalentlar) yangi turlarni qo'shishga tayyor; javob tekshirish funksiyasi (3.06 va 3.08 qayta ishlatadi); filterli va paginatsiyali Owner API; Learner uchun faqat published va answer keysiz read API; source/license majburiy maydon; Owner CMS: ro'yxat, filterlar, editor (MCQ/numeric), formula preview, versiya tarixi; unit, integratsiya va komponent testlari.

## 3.04-Learner · Lessons — 2026-10-11
Status: Blocked by 3.03 (u esa 0.00, 0.01, 0.02, 1.02, 3.01, 3.02, 4.01 ga bloklangan); shuningdek 1.01, 2.03
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `docs/API.md`, `docs/DB.md`, `docs/ARCHITECTURE.md` hali mavjud emas. PRD §12.3 va PROGRESS'dagi 3.03 yozuvi o'qildi.
Testlar: —
Muammolar: Learner lesson sahifasi (title, learning objectives, explanation, formula va qoidalar, worked examples, practice questions, related topics, bookmark, lesson progress) quyidagilarga tayanadi:
- 3.03: lessons va `lesson_progress` jadvallari hamda Learner lesson API (published lesson olish, progress start/update/complete). Juft sessiya qoidasiga ko'ra 3.04 backend/DB yozmaydi, faqat `docs/API.md` dagi 3.03 kontraktidan foydalanadi.
- 0.00: `web/` tuzilmasi. 0.02: UI kit va tokenlar (KaTeX/formula rendering 3.03 kontent formati bilan kelishiladi).
- 1.01: learner auth (sessiya, himoyalangan route'lar). 2.03: learner layout va sidebar.
- 3.02: domain/topic katalogi (lessonga topic orqali kirish, related topics havolalari).
- Qisman: 3.05/3.06 (practice questions bloki — tayyor bo'lmasa blok ko'rsatilmaydi, fake savol yo'q), 2.08 (lesson bookmark modeli — bookmark kimga tegishli ekani kelishiladi).
Keyingi qadam: 3.03 tugab `docs/API.md` ga lesson kontraktini yozgach, 3.04 ni boshlash. Avval reja beriladi: lesson sahifasi route'i (`/learn/<domain>/<topic>/<lesson>` yoki `/lessons/<slug>`), structured blok renderer (heading/paragraph/list/formula/image/worked_example/callout) XSS'siz, progress tracking (start, scroll/oxirgi pozitsiya, "completed" belgisi), loading/empty/error statelar, unit va komponent testlari.

## 3.03-Owner · Lesson Editor — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 3.01, 3.02, 4.01 (related questions uchun qisman 3.05)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md`, `docs/API.md`, `docs/DB.md` hali mavjud emas. PRD §22 (22.2 Lesson Editor), §12.3 (Lesson sahifasi) va §3 ("Lessons" entity) o'qildi.
Testlar: —
Muammolar: Lesson Editor (lessons backend, API, DB va Owner editor: title, learning objectives, rich text yoki structured content, headings, lists, formulas, images, worked examples, related questions, related topics, preview, Save Draft, Publish) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware), **storage interfeysi** (lesson rasmlari uchun), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool. `lessons` (va kerak bo'lsa `lesson_blocks`, `lesson_related_questions`, `lesson_related_topics`) jadvallarini 3.03 o'z migrationi bilan qo'shadi. `lesson_progress` jadvali 3.04 (Learner) ga kerak, lekin juft sessiya qoidasiga ko'ra DB/API'ni 3.03 quradi.
- 0.02: design tokenlar va UI kit (form, tabs, toolbar, modal, image uploader, preview frame).
- 1.02: owner auth, RBAC middleware va audit helper (create/update/publish/archive audit logga yoziladi).
- 3.01: Draft/In Review/Published/Archived workflow, CMS skeleti va preview mexanizmi. 3.03 o'z workflow'ini yasamaydi, lessons bo'limini 3.01 skeletiga ulaydi.
- 3.02: domain/topic/subtopic taksonomiyasi (lesson → topic FK, related topics).
- 4.01: owner layout va sidebar.
- 3.05: `questions` jadvali. Related questions shu jadvalga FK bilan bog'lanadi; 3.05 hali tayyor bo'lmasa, related questions maydoni yoqilmaydi (fake savol yozilmaydi).
- Iste'molchilar: 3.04 (Learner lesson sahifasi va progress) faqat 3.03 API'sidan foydalanadi; 2.04 (Continue Learning), 2.05 (lesson vazifalari), 2.06 (AI kontekst), 2.08 (lesson bookmark), 4.05 (AI draft lesson) lesson ID va kontraktiga tayanadi.
Eslatma: oldingi PROGRESS yozuvlarining ayrimlari eski raqamlashdan foydalanadi (masalan "3.04: questions", "3.05: practice", "3.11: AI Core"). Joriy CLAUDE.md xaritasida 3.04 = Learner Lessons, 3.05 = Question Bank, 3.06 = Daily Practice, 3.12 = AI Core.
Keyingi qadam: 0.00 → 0.01 → 0.02 → 1.01 → 1.02 → 2.03 → 4.01 → 3.01 → 3.02 tugagach, 3.03 ni boshlash. Avval reja beriladi va tasdiq olinadi: kontent formati (structured JSON bloklar — heading/paragraph/list/formula/image/worked_example/callout — vs Markdown + KaTeX; tavsiya: versiyalanadigan JSON bloklar, server-side schema validatsiya va XSS sanitizatsiya), formula rendering (KaTeX, LaTeX manbasi saqlanadi), editor kutubxonasi (TipTap vs o'zimizning blok editor; yangi dependency tasdiq bilan), Published lesson tahriri (yangi draft versiya yaratib, publish'da almashtirish vs to'g'ridan-to'g'ri — 3.01 workflow bilan kelishiladi), autosave/optimistic locking (`version` ustuni bilan konfliktni aniqlash), rasm limitlari (format, hajm, re-encode), `slug` va tartib (`sort_order` topic ichida), learning objectives alohida maydonmi, `lesson_progress` modeli (started/completed, oxirgi pozitsiya) 3.04 bilan kelishiladi.

## 3.02-Learner & Owner · Curriculum — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 1.02, 2.03, 3.01, 4.01
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §12 (12.1–12.2) va bog'liq §3 ("Curriculum Domains, Topics" entity'lari) va §22 (CMS) o'qildi.
Testlar: —
Muammolar: Curriculum (rasmiy 8 ta SAT domain: Math — Algebra, Advanced Math, Problem-Solving and Data Analysis, Geometry and Trigonometry; R&W — Information and Ideas, Craft and Structure, Expression of Ideas, Standard English Conventions; har bir domain ichida topic va subtopiclar; Owner CMS'da boshqarish; Learner katalogi) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool. `curriculum_domains`, `topics` (subtopic uchun `parent_id` yoki alohida jadval) jadvallarini 3.02 o'z migrationi bilan qo'shadi va 8 domainni seed qiladi.
- 0.02: design tokenlar va UI kit (accordion/tree, list, badge, empty state, drag-handle).
- 1.01: learner auth middleware (katalog endpointlari). 1.02: owner RBAC middleware va audit helper (har bir create/update/reorder/status o'zgarishi audit logga yoziladi).
- 3.01: Draft/In Review/Published/Archived workflow va CMS skeleti — curriculum bo'limi shu skeletga ulanadi, Learner faqat Published topiclarni ko'radi.
- 2.03 / 4.01: learner va owner layoutlari (sidebar havolalari).
- Iste'molchilar: 3.03 (lesson → topic), 3.05 (savol → domain/topic), 3.07 (domain distribution), 3.10, 3.11, 2.05, 2.07, 2.08, 2.09, 4.04 domain/topic ID'lariga FK orqali tayanadi. Shuning uchun taksonomiya kontrakti barqaror bo'lishi kerak.
Keyingi qadam: 0.00 → 0.01 → 0.02 → 1.01 → 1.02 → 2.03 → 4.01 → 3.01 tugagach, 3.02 ni boshlash. Avval reja beriladi va tasdiq olinadi: 8 domain qat'iy (seed, Owner faqat nom/tavsif/tartibni tahrirlaydi) yoki to'liq CRUD; subtopic ierarxiyasi (bitta `topics` jadvali + `parent_id`, maksimal chuqurlik 2) vs alohida `subtopics` jadvali; stabil `slug` (URL va AI/import uchun); tartiblash (`sort_order` + reorder endpointi); topic arxivlanganda unga bog'langan lesson/savollar bilan nima bo'ladi (soft delete, hard delete taqiqlanadi); Learner katalogida topic bo'yicha progress ko'rsatilishi 3.04/2.07 bilan kelishiladi.

## 3.01-Learner & Owner · Content CMS — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.02, 4.01 (Learner tomoni uchun qisman 1.01)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §22 (22.1 Status, 22.3 Navigation; 22.2 Lesson Editor 3.03 ga tegishli) o'qildi.
Testlar: —
Muammolar: Content CMS (Draft → In Review → Published → Archived workflow, CMS skeleti va accordion navigatsiya, preview; Learner faqat Published kontentni ko'radi) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi. Umumiy workflow paketi (status enum, ruxsat etilgan o'tishlar, published-only filter helper) shu tuzilmaga joylashadi.
- 0.01: migration tool va `audit_logs` jadvali. Status ustuni/umumiy jadval 3.01 o'z migrationi bilan qo'shiladi.
- 0.02: design tokenlar va UI kit (accordion, status badge, tabs, confirm dialog, preview frame).
- 1.02: owner auth, RBAC middleware (kim publish/approve qila oladi) va audit yozish helperi (har bir status o'tishi audit logga yoziladi).
- 4.01: owner layout va sidebar — CMS sahifasi shu layout ichida, sidebar'dagi "Content" havolasi.
- 1.01: Learner tomonidagi published-only endpointlar auth talab qilsa, learner middleware.
- Iste'molchilar: 3.02 (curriculum), 3.03 (lessons), 3.05 (questions), 3.07 (mock exams), 3.10 (formulas), 4.05 (AI drafts) shu workflow va CMS skeletiga o'z bo'limlarini ulaydi. Shuning uchun kontrakt ular uchun qayta ishlatiladigan bo'lishi kerak; 3.01 ularning editorlarini yasamaydi.
Keyingi qadam: 0.00 → 0.01 → 0.02 → 1.01 → 1.02 → 2.03 → 4.01 tugagach (tavsiya etilgan tartibda 2.02 dan keyin), 3.01 ni boshlash. Avval reja beriladi va tasdiq olinadi: workflow modeli (har bir kontent jadvalida `status` + umumiy Go paket vs polimorf `content_items` jadvali), ruxsat etilgan o'tishlar (Published → Draft qaytadimi yoki Published tahriri yangi versiya/draft nusxa yaratadimi — 3.05/3.07 versiyalash bilan kelishiladi; Archived → Draft tiklash), "In Review" ni kim tasdiqlaydi (1.02 rollari; bitta owner bo'lsa review majburiymi), preview usuli (owner-only preview route vs muddatli preview token), status o'tishlari tarixi (`content_status_history` jadvali vs faqat audit log), scheduled publish kerakmi, accordion holati (ochiq bo'limlar localStorage'da saqlanadimi).

## 2.11-Learner · Settings & Appearance — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 2.03 (Learning Preferences uchun 2.02, 2.05; Notifications uchun 3.13; Subscription tab uchun 3.12)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §4.1–4.6, §20 va §31 o'qildi.
Testlar: —
Muammolar: Settings & Appearance (Appearance: Light/Dark/System, accent, wallpaper gallery + upload/almashtirish/olib tashlash, Light va Dark uchun alohida wallpaper, opacity/overlay, blur, sidebar ko'rinishi, compact/comfortable, motion; Learning Preferences; Notification sozlamalari; login qilgan userda DB'da saqlash, guest'da local, o'zgarish darhol qo'llanishi) quyidagilarga tayanadi:
- 0.00: Go core (config, error handler, middleware, rate limiter), **storage interfeysi** (wallpaper upload), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool va `user_preferences` jadvali. 2.11 qo'shimcha ustunlar/`user_wallpapers` jadvalini o'z migrationi bilan qo'shadi, `user_preferences` ni qayta yaratmaydi.
- 0.02: design tokenlar, theme provider (light/dark/system), accent mexanizmi va UI kit (tabs, segmented control, slider, switch, color swatch). 2.11 theme/accent tizimini o'zi yasamaydi, faqat 0.02 ning API'si orqali tanlovni saqlaydi va qo'llaydi. Compact/comfortable va reduced-motion 0.02 tokenlarida (density, transition) qo'llab-quvvatlanishi kerak.
- 1.01: learner auth middleware (DB'da saqlash), guest/logged-in farqi va login paytida local preference'ni serverga ko'chirish.
- 2.03: learner layout — wallpaper qo'llanadigan joy (dashboard kabi sahifalar), sidebar collapsed holati (localStorage vs `user_preferences`, 2.03 bilan kelishiladi), exam layout'da wallpaper/effektlar o'chirilishi (3.07 bilan).
- Learning Preferences: target score, exam date, daily time, study days 2.02 da yaratiladi; 2.11 shu maydonlarni tahrirlaydi va o'zgarganda 2.05 ning qayta hisoblash hook'ini chaqiradi.
- Notifications: 3.13 bildirishnoma turlari va kanallari (email/in-app) modeli. 2.11 faqat toggle UI va preference saqlashni qiladi, yuborish logikasini yasamaydi.
- Subscription tab: 3.12 ga tegishli, 2.11 faqat Settings ichida joy/havola qoldiradi.
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.01 → 2.03 tugagach (tavsiya etilgan tartibda 2.10 dan keyin), 2.11 ni boshlash. Avval reja beriladi va tasdiq olinadi: preferences saqlash shakli (alohida ustunlar vs `appearance` JSONB + schema validatsiya), accent palitrasi (0.02 bilan, nechta rang), wallpaper limitlari (format: JPEG/PNG/WebP, max hajm, max o'lcham, server-side re-encode va EXIF tozalash, user boshiga nechta custom wallpaper), standart galereya qayerda saqlanadi (static asset vs storage), SSR'da theme flash'ning oldini olish (cookie vs inline script), guest → login bo'lganda local va server sozlamalari to'qnashsa qaysi biri ustun, saqlash debounce/optimistic update.

## 2.10-Learner · My Profile — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 2.01, 2.03 (deletion cascade uchun qisman 3.12, 5.01)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §20 Account (va bog'liq §6.1, §7, §28) o'qildi.
Testlar: —
Muammolar: My Profile (profil tahriri: ism, avatar, telefon, DOB; email o'zgartirish faqat yangi emailga verification code orqali; password o'zgartirish/qo'shish; login usullari: Google ulash/uzish, oxirgi login usulini o'chirishga yo'l qo'ymaslik; account deletion) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware, rate limiter), **storage interfeysi** (avatar), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool, `users`/`profiles` jadvallari va ERD (deletion cascade/soft delete strategiyasi shu yerda belgilanadi).
- 0.02: design tokenlar va UI kit (form, input, avatar, modal/confirm dialog).
- 1.01: session modeli, password hashing, email verification code mexanizmi (email o'zgartirishda qayta ishlatiladi, 2.10 o'zi yasamaydi), Google OAuth (account linking uchun), sessionlarni bekor qilish (password/email o'zgarganda va deletion'da).
- 2.01: profil maydonlari va avatar upload logikasi. 2.10 ularni tahrirlash uchun qayta ishlatadi, qayta yozmaydi.
- 2.03: learner layout, sidebar'dagi Account guruhi va header profile dropdown'dagi "My Profile" havolasi.
- 3.12/5.01: account deletion'da faol obunani bekor qilish (to'lov provayderi bilan). 2.10 faqat 3.12 tomonidan beriladigan hook/servisni chaqiradi.
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.01 → 2.01 → 2.03 tugagach (tavsiya etilgan tartibda 2.04 dan keyin), 2.10 ni boshlash. Avval reja beriladi va tasdiq olinadi: account deletion turi (darhol hard delete vs soft delete + N kunlik grace period va keyin purge job), deletion tasdiqlash usuli (password yoki email code, Google user uchun re-auth), anonimlashtirilgan ma'lumotlar (analytics, audit log) saqlanadimi, email o'zgartirishda eski emailga ogohlantirish yuboriladimi, password o'zgarganda boshqa sessionlar chiqariladimi, Google ulash/uzish shartlari (password o'rnatilmagan bo'lsa uzib bo'lmaydi), sensitive amallar uchun rate limit.

## 2.09-Learner · Mistake Notebook — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 2.03, 3.02, 3.04, 3.05 (exam xatolari uchun 3.07/3.08)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §17 (va bog'liq §3 "Mistake Records" entity, §10.2 Track guruhi) o'qildi.
Testlar: —
Muammolar: Mistake Notebook (noto'g'ri javoblar avtomatik qo'shiladi; yozuvda savol, learner javobi, to'g'ri javob, explanation, mavzu, xato sanasi, qayta urinishlar, "o'zlashtirdim" belgisi; spaced repetition'ga tayyor model) quyidagilarga tayanadi:
- 0.00: Go core, `api/internal/<module>/` va `web/` tuzilmasi. 0.01: migration tool va `users` (`mistake_records` va `mistake_attempts` jadvallarini 2.09 o'z migrationi bilan qo'shadi).
- 0.02: design tokenlar va UI kit (list/card, filter, badge, empty state). 1.01: learner auth middleware. 2.03: learner layout va sidebar'dagi "Mistake Notebook" havolasi.
- 3.04: `questions`/`question_versions` (yozuv aniq versiyaga bog'lanadi, chunki learner javobi va explanation o'sha versiyaga tegishli). 3.02: topic/domain (filter va 2.04 Weak Topics uchun).
- 3.05: javobni tekshirish nuqtasi. "Avtomatik qo'shish" 3.05 dagi answer checking'dan keyin chaqiriladigan hook/service orqali bo'ladi (minimal integratsiya). Qayta urinishda javob tekshirish va explanation logikasi 3.05 dan qayta ishlatiladi, 2.09 o'zi yasamaydi.
- 3.07/3.08: mock exam xatolari ham notebookga tushadimi — shu sessiyalar bilan kelishiladi (submit'dan keyin batch qo'shish).
- Iste'molchilar: 2.04 (Weak Topics), 2.05 (review vazifalari), 2.06 (xato sababini tushuntirish), 2.07 (xatolar va takroriy xatolar soni) 2.09 API'sini o'qiydi.
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.01 → 2.03 → 3.02 → 3.04 → 3.05 tugagach (tavsiya etilgan tartibda 2.08 dan keyin), 2.09 ni boshlash. Avval reja beriladi va tasdiq olinadi: unique constraint (`user_id`, `question_id`) — bir savoldagi takroriy xato yangi yozuvmi yoki mavjudiga `wrong_count++`; "o'zlashtirdim" qo'lda belgilanadimi yoki N ta ketma-ket to'g'ri javobdan keyin avtomatikmi; o'zlashtirilgan savolda yana xato qilinsa yozuv qayta ochiladimi; spaced repetition maydonlari (`next_review_at`, `interval_days`, `ease_factor`, `repetitions` — SM-2 ga tayyor, lekin hozircha scheduler yoqilmaydi); manba (`practice` / `mock_exam` / `placement`); filterlar (domain, topic, holat, manba, sana) va pagination.

## 2.08-Learner · Saved Questions — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 2.03, 3.02, 3.04 (bookmark tugmasi va qayta ishlash uchun 3.05)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §17 (va bog'liq §3 "Bookmarks" entity, §10.2, §12.3, §12.4, §18) o'qildi.
Testlar: —
Muammolar: Saved Questions (saqlanganlar ro'yxati, topic bo'yicha filter, savolni ochish, bookmarkni olib tashlash, qayta ishlash) quyidagilarga tayanadi:
- 0.00: Go core, `api/internal/<module>/` va `web/` tuzilmasi. 0.01: migration tool va `users` jadvali (`bookmarks` jadvalini 2.08 o'z migrationi bilan qo'shadi).
- 0.02: design tokenlar va UI kit (list/card, filter select, bookmark icon button, empty state).
- 1.01: learner auth middleware. 2.03: learner layout va sidebar'dagi "Saved Questions" havolasi (Track guruhi).
- 3.04: `questions` va `question_versions`. Bookmark savolga FK bilan bog'lanadi; savol arxivlansa yoki yangi versiya chiqsa nima ko'rsatilishi 3.04 modeliga bog'liq.
- 3.02: domain/topic taksonomiyasi (topic bo'yicha filter uchun).
- 3.05: practice/review ekranidagi bookmark tugmasi va "qayta ishlash" oqimi (saqlangan savollardan practice set yaratish yoki bitta savolni practice rejimida ochish). 2.08 javob tekshirish va explanation logikasini o'zi yasamaydi, 3.05 dan qayta ishlatadi. Mock exam review'dagi (3.08) bookmark ham shu endpointni chaqiradi.
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.01 → 2.03 → 3.02 → 3.04 → 3.05 tugagach (tavsiya etilgan tartibda 3.05 dan keyin), 2.08 ni boshlash. Avval reja beriladi va tasdiq olinadi: `bookmarks` faqat savollar uchunmi yoki umumiy polimorf jadval (savol, lesson §12.3, formula §18; 3.03 va 3.09 bilan kelishiladi), bookmark `question_id` ga yoki aniq `question_version_id` ga bog'lanadi, arxivlangan/o'chirilgan savol ro'yxatda qanday ko'rinadi, unique constraint (`user_id`, `question_id`), filterlar (topic, domain, qiyinlik, savol turi, saqlangan sana) va pagination (cursor vs offset), ixtiyoriy shaxsiy izoh (note) kerakmi, "qayta ishlash" bitta savolmi yoki tanlanganlardan practice set.

## 2.07-Learner · My Progress — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 2.03 (metrikalar manbasi uchun 3.02, 3.05, 3.07, 3.08, 2.05, 2.09)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §16 (va bog'liq §10.4, §15, §17) o'qildi.
Testlar: —
Muammolar: My Progress sahifasi "faqat real saqlangan ma'lumotlardan" hisoblanadi (§16), shuning uchun har bir metrika manba modulga tayanadi:
- 0.00: Go core, `api/internal/<module>/` va `web/` tuzilmasi. 0.01: migration tool, `users`. 0.02: design tokenlar, UI kit va chart komponentlari (yoki chart kutubxonasi tanlovi). 1.01: learner auth middleware. 2.03: learner layout va sidebar'dagi "My Progress" havolasi.
- Practice activity, accuracy, o'rtacha javob vaqti, domain mastery: 3.05 (practice attempt/answer yozuvlari, `time_spent`), 3.02 (domain/topic taksonomiyasi).
- Mock exam tarixi, Math va R&W progressi: 3.07 (attemptlar), 3.08 (scaled/taxminiy natija, domain breakdown).
- Study plan completion: 2.05 (`study_plan_tasks` holati).
- Xatolar va takroriy xatolar soni: 2.09 (mistake yozuvlari va qayta urinishlar).
- Haftalik o'qish davomiyligi va vaqt bo'yicha grafik: practice/lesson/exam sessiyalari vaqti (3.03, 3.05, 3.07).
Manba modul hali qurilmagan metrika uchun empty state ko'rsatiladi, fake raqam yozilmaydi. 2.04 (Math/R&W progress, Recent Activity, streak bloklari) 2.07 API'sini o'qiydi, shuning uchun kontrakt unga mos bo'lishi kerak.
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.01 → 2.03 tugagach, tavsiya etilgan tartibda (3.05, 3.08, 2.09, 2.05 dan keyin) 2.07 ni boshlash. Avval reja beriladi va tasdiq olinadi: metrikalar har so'rovda hisoblanadimi yoki agregat jadval/materialized view (`learner_daily_stats`) bilan, kun/hafta chegarasi qaysi timezone bo'yicha, domain mastery formulasi (accuracy + minimal savollar soni + oxirgi urinishlar vazni), grafik vaqt oraliqlari (7/30/90 kun), unified `learner_activity` jurnali kerakmi (2.04 Recent Activity va streak uchun ham).

## 2.06-Learner · AI Study Assistant — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 2.03, 3.11 (kontekst uchun qisman 3.03, 3.04, 3.09, 2.09, 3.12)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §19 (va bog'liq §23.3, §3 "AI Conversations" entity) o'qildi.
Testlar: —
Muammolar: AI Study Assistant (learner chat; savolni tushuntirish, bosqichma-bosqich yechim, formulani izohlash, xato javob sababini tushuntirish, o'xshash savol va qo'shimcha misol; platforma savol/lesson ma'lumotidan foydalanish; usage va rate limitlar) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware, rate limiter), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool va `users` jadvali (`ai_conversations`/`ai_messages` jadvallarini 2.06 o'z migrationi bilan qo'shadi).
- 0.02: design tokenlar va UI kit (chat bubble, textarea, markdown/math rendering uchun komponentlar).
- 1.01: learner auth middleware/session.
- 2.03: learner layout (sidebar'dagi "AI Study Assistant" havolasi shu layout ichida).
- 3.11: **AI Core**: provider client, API kalitlari (`.env`, frontendga chiqmaydi), usage limitlar, rate limit, cost control, loglar. 2.06 provider'ni to'g'ridan-to'g'ri chaqirmaydi, faqat 3.11 interfeysi orqali ishlaydi.
- Kontekst manbalari: savol va explanation (3.04), lesson (3.03), formula (3.09), xato yozuvi (2.09). Ular hali yo'q bo'lsa, "savolni tushuntir" kabi kontekstli tugmalar o'sha modul tayyor bo'lgunicha yoqilmaydi. Erkin chat esa 3.11 tayyor bo'lsa ishlaydi.
- 3.12: plan bo'yicha AI limitlari (feature gating backendda).
Keyingi qadam: 0.00 → 0.01 → 0.02 → 1.01 → 2.03 → 3.11 tugagach, 2.06 ni boshlash (tavsiya etilgan tartibda 3.11 dan keyin). Avval reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi: javob streaming (SSE) bo'ladimi yoki oddiy response, suhbat tarixi qancha saqlanadi va learner uni o'chira oladimi, AI yaratgan "o'xshash savol" learnerga darhol ko'rsatiladimi yoki "AI yaratgan, tasdiqlanmagan" belgisi bilanmi (PRD: muhim kontentni Owner tasdiqlaydi), matematik yechim qanday tekshiriladi (3.11 dagi validator), math rendering kutubxonasi (KaTeX).

## 2.05-Learner · My Study Plan — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 2.03 (algoritm kirishlari uchun 2.02, 3.10; vazifa bog'lanishlari uchun 3.02, 3.03, 3.05, 2.09)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §9.2, §11 (va bog'liq §8, §9.1, §10.4, §16) o'qildi.
Testlar: —
Muammolar: My Study Plan (rule-based reja algoritmi, daily va weekly view, vazifa davomiyligi va bajarilgan/bajarilmagan holati, vazifani boshlash/davom ettirish, rejani qayta hisoblash, kunlik vaqt va imtihon sanasini o'zgartirish) quyidagilarga tayanadi:
- 0.00: Go core, `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool, `users`/`user_preferences`. `study_plans` va `study_plan_tasks` jadvallarini 2.05 o'z migrationi bilan qo'shadi.
- 0.02: design tokenlar va UI kit (kalendar/hafta ko'rinishi, task kartasi, progress).
- 1.01: learner auth middleware. 2.03: learner layout va sidebar'dagi "My Study Plan" havolasi.
- Algoritm kirishlari: maqsadli ball, imtihon sanasi, kunlik vaqt, o'qish kunlari (2.02); domain natijalari va diagnostik profil (3.10). 3.10 bo'lmasa, "testsiz boshlash" yo'li (§9 yo'l 3) uchun default domain tartibi ishlatiladi, fake natija yozilmaydi.
- Vazifa bog'lanishlari: lesson (3.03), practice set (3.05), review sessiyasi (2.09 mistake'lar), topic/domain (3.02). Manba hali bo'lmasa, o'sha turdagi vazifa yaratilmaydi.
- 2.04 (Today's Study Plan bloki) va 2.07 (study plan completion) 2.05 ma'lumotini o'qiydi, shuning uchun API ularga mos bo'lishi kerak.
Keyingi qadam: kamida 0.00 → 0.01 → 0.02 → 1.01 → 2.03 → 2.02 tugagach (tavsiya etilgan tartibda 3.10 dan keyin), 2.05 ni boshlash. Avval reja beriladi va tasdiq olinadi: algoritm qoidalari (domain vazni = zaiflik × SAT ulushi, lesson/practice/review nisbati, haftalik mock exam bormi), qayta hisoblash triggerlari (preferences o'zgarishi, o'tkazib yuborilgan kunlar, yangi diagnostik/mock natija), o'tkazib yuborilgan vazifalar keyingi kunga ko'chadimi yoki "missed" bo'lib qoladimi, kun chegarasi qaysi timezone bo'yicha, plan versiyalash (eski vazifalar va natijalar saqlanadi, faqat kelajakdagilar qayta yoziladi).

## 2.02-Learner · Onboarding — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01, 2.01
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §8 (va bog'liq §7, §9) o'qildi.
Testlar: —
Muammolar: Onboarding (maqsadli ball, imtihon sanasi, ixtiyoriy sabab; kunlik vaqt, haftalik kunlar, Math/R&W ishonch darajasi, oldingi SAT natijasi; progress indicator, Back/Continue/Skip, har bir qadamni serverda saqlash va keyin tahrirlash, oxirida Placement Assessment sahifasiga o'tish) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool, `users`/`profiles`/`user_preferences` jadvallari (onboarding javoblari va `onboarding_step`/`onboarding_completed_at` holati 2.02 o'z migrationi bilan qo'shiladi).
- 0.02: design tokenlar va UI kit (stepper/progress, select, radio, slider, date picker).
- 1.01: learner auth middleware/session.
- 2.01: Profile setup tugagach onboarding'ga yo'naltirish va route guard (profile → onboarding → placement).
- 3.10: Placement Assessment sahifasi hali yo'q — 2.02 faqat unga redirect qiladi (route nomi 3.10 bilan kelishiladi), sahifaning o'zini yasamaydi.
Keyingi qadam: 0.00 → 0.01 → 0.02 → 1.01 → 2.01 tugagach, 2.02 ishini boshlash. Avval reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi: maqsadli ball diapazoni va qadami (400–1600, 10 qadam), imtihon sanasi uchun rasmiy SAT sanalari ro'yxatimi yoki erkin sana, kunlik vaqt variantlari (masalan 15/30/45/60/90+ daqiqa), ishonch darajasi shkalasi (1–5), oldingi natija formati (total + Math/R&W bo'limlari), ma'lumot `user_preferences` da saqlanadimi yoki alohida `onboarding_responses` jadvalida (2.05 study plan shu ma'lumotni o'qiydi).

## 2.01-Learner · Profile setup — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02, 1.01
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §7 o'qildi.
Testlar: —
Muammolar: Profile setup (avatar upload + standart avatar, first/last name, read-only tasdiqlangan email, ixtiyoriy telefon, tug'ilgan sana/yosh, email/password account uchun password + confirmation, Google ma'lumotlarini oldindan to'ldirish, tugagach onboarding'ga o'tish) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware), **storage interfeysi** (avatar upload uchun), `api/internal/<module>/` va `web/` tuzilmasi.
- 0.01: migration tool, `users` va `profiles` jadvallari (profile maydonlari va `profile_completed` kabi holat 2.01 o'z migrationi bilan qo'shiladi).
- 0.02: design tokenlar va UI kit (form, input, avatar komponentlari).
- 1.01: learner session/auth middleware, email verification holati, Google OAuth profil ma'lumotlari, password hashing helperi.
Keyingi qadam: 0.00 → 0.01 → 0.02 → 1.01 tugagach, 2.01 ishini boshlash. Avval reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi: DOB vs yosh oralig'i (minimal yosh cheklovi bormi), avatar formatlari/hajm limiti va resize, telefon formati (E.164), Google foydalanuvchisi uchun password ixtiyoriymi.

## 1.02-Auth · Owner — 2026-10-11
Status: Blocked by 0.00, 0.01, 0.02 (va qisman 1.01)
Bajarildi: Hech narsa. Repo tekshirildi: `api/`, `web/`, `api/migrations/` va `docs/ARCHITECTURE.md` hali mavjud emas. PRD §6.2, §21.1, §28 o'qildi.
Testlar: —
Muammolar: Owner auth (`/owner/login`, MFA, RBAC middleware, audit yozish helperi, login rate limiting, owner seed/admin CLI) quyidagilarga tayanadi:
- 0.00: Go core (config, logger, error handler, middleware zanjiri, rate limiter), `api/internal/<module>/` tuzilmasi, `web/` tuzilmasi.
- 0.01: migration tool, `users` (role maydoni) va `audit_logs` jadvallari. Owner MFA va owner sessionlari uchun jadvallarni 1.02 o'z migrationlari bilan qo'shadi.
- 0.02: design tokenlar va UI kit (login va MFA formalari uchun).
- 1.01: session modeli (cookie + DB session vs JWT) va password hashing umumiy bo'lishi kerak. 1.01 avval bajarilsa, 1.02 uning session infratuzilmasini qayta ishlatadi.
Keyingi qadam: 0.00 → 0.01 → 0.02 → 1.01 tugagach, 1.02 ishini boshlash. Avval reja beriladi va quyidagi qarorlar bo'yicha tasdiq olinadi: MFA turi (TOTP + recovery codes tavsiya etiladi), owner session muddati va idle timeout, RBAC rollari (masalan `owner`, `editor`, `viewer` yoki faqat `owner`), birinchi owner'ni yaratish usuli (seed vs CLI buyruq), audit log formati.

## 0.03-Landing (all pages) — 2026-10-11
Status: Blocked by 0.00, 0.02
Bajarildi: Hech narsa. Repo tekshirildi: `web/` (Next.js + Tailwind), `docs/ARCHITECTURE.md` va design system (tokenlar, theme, UI kit) hali mavjud emas. PRD §5 o'qildi.
Testlar: —
Muammolar: Landing sahifalari (Landing, Pricing va FAQ bo'limlari, Privacy, Terms, Contact, SEO) 0.00 belgilaydigan `web/` tuzilmasiga va 0.02 beradigan design token/UI komponentlarga tayanadi. Tokenlarsiz yozilsa, CLAUDE.md dagi "faqat design tokenlar" qoidasi buziladi.
Keyingi qadam: 0.00 va 0.02 tugagach, 0.03 ishini boshlash. Avval sahifalar va bo'limlar rejasini berib, tasdiq olish.
