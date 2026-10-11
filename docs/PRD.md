# SAT SELF-STUDY PLATFORM — PRD va Technical Specification

- Loyiha turi: SAT imtihoniga mustaqil tayyorlanish platformasi
- Asosiy foydalanuvchilar: SAT topshirishga tayyorlanayotgan learnerlar
- Boshqaruv tizimi: Owner Dashboard
- Tillar: ingliz, o'zbek va rus tillari uchun kengaytirishga tayyor arxitektura
- Dizayn: minimalistic, professional, responsive, foydalanuvchiga moslashtiriladigan
- Texnologiyalar: Next.js, React, TypeScript, Tailwind CSS, Go, Gin, PostgreSQL

## 1. Loyihaning asosiy maqsadi
SAT'ga tayyorlanayotgan foydalanuvchilar mustaqil o'qishi, mavzularni o'rganishi, savollar ishlashi, xatolarini tahlil qilishi va real SAT formatidagi mock exam topshirishi mumkin bo'lgan to'liq platforma. Bu oddiy test sayti emas. U learnerga quyidagilarda yordam beradi:
- Hozirgi bilim darajasini aniqlash.
- Maqsadli SAT ballini belgilash.
- Shaxsiy o'quv rejasini shakllantirish.
- Har kuni reja asosida o'qish.
- Math va Reading & Writing bo'yicha mavzularni o'rganish.
- Practice savollarini ishlash.
- Noto'g'ri javoblarni tushunish va qayta ishlash.
- Vaqt chegaralangan mock exam topshirish.
- O'z natijalari va rivojlanishini kuzatish.
- Tushunmagan mavzularini AI yordamida o'rganish.

Platforma rasmiy SAT sertifikatini bermaydi. U foydalanuvchini rasmiy imtihonga tayyorlaydi.

## 2. Loyihaning asosiy qismlari
### 2.1. Learner taraf
Landing Page, Authentication, Profile Setup, Onboarding, Placement Test, Study Plan, Learner Dashboard, Daily Practice, Curriculum va Lessons, Mock Exams, Exam Results, My Progress, Saved Questions, Mistake Notebook, Formula & Concept Library, AI Study Assistant, Profile va Settings, Subscription va Billing.

### 2.2. Owner taraf
Owner Authentication, Owner Dashboard, Curriculum CMS, Lesson Editor, Question Bank CMS, Mock Exam Builder, AI Content Assistant, Content Preview va Publishing, Learner Management, Platform Analytics, Subscription Management, Platform Settings, Audit Logs.

Learner va Owner interfeyslarining layout, navigation va authorization qoidalari alohida. Ikkalasi bitta backend va bitta PostgreSQL bazasi bilan ishlaydi.

## 3. Texnologik talablar
**Frontend:** Next.js va React, TypeScript, Tailwind CSS, qayta ishlatiladigan UI komponentlari, responsive dizayn, accessible formalar va klaviatura navigatsiyasi, Light va Dark Mode, foydalanuvchi sozlamalarini saqlash, API bilan xavfsiz aloqa.

**Backend:** Go, Gin, REST API, modullarga ajratilgan arxitektura, input validation, markazlashtirilgan error handling, authentication va authorization middleware, structured logging, unit va integration testlar.

**Database:** PostgreSQL. Asosiy entity'lar: Users, Profiles, User Preferences, Learner Goals, Study Plans, Study Plan Tasks, Curriculum Domains, Topics, Lessons, Questions, Question Versions, Practice Attempts, Mock Exams, Exam Modules, Exam Attempts, Exam Responses, Bookmarks, Mistake Records, AI Conversations, Subscriptions, Payments, Notifications, Audit Logs. Sxemada relationship, foreign key, unique constraint, index va migrationlar to'g'ri tashkil qilinsin.

**Muhim texnik qoida:** Frontend va backend bitta repo ichida bo'lishi mumkin, lekin mas'uliyatlari ajratilgan bo'lsin: `web/` (Next.js), `api/` (Go), `docs/`, migrationlar uchun `database/` yoki `api/migrations/`. Mavjud repository strukturasini tekshirmasdan o'chirma va qayta yaratma.

## 4. Dizayn tizimi
### 4.1. Asosiy tamoyil
Default dizayn minimalistic, toza, professional va ko'zga yoqimli bo'lsin. Keraksiz ranglar, ortiqcha gradientlar, katta bezaklar, haddan tashqari ko'p kartalar va chalg'ituvchi animatsiyalar bo'lmasin. Dizayn zamonaviy education SaaS mahsulotlariga mos bo'lsin. Talablar: aniq visual hierarchy, o'qilishi oson typography, bir xil spacing, izchil border radius, minimal va foydali animatsiyalar, tushunarli hover/focus/loading/disabled holatlari, desktop/tablet/mobile moslashuvi, yaxshi kontrast va accessibility. Glassmorphism faqat haqiqiy qiymat qo'shadigan joyda ishlatiladi.

### 4.2. Default theme
Birinchi ishga tushganda neytral ranglardagi Light Mode ochiladi. Dark Mode ham to'liq ishlab chiqiladi. Theme o'zgarganda barcha sahifalar, modal, dropdown, input, tooltip va chartlar yangilanishi kerak, faqat fon rangi emas.

### 4.3. Design tokens
Markazlashtirilgan tokenlar: primary/secondary, background, surface, text, border, success/warning/error/info ranglari, typography scale, spacing scale, border radius, shadows, z-index, transition duration. Tasodifiy rang yoki spacing qiymatlari ishlatilmasin.

### 4.4. Foydalanuvchi moslashtirishi
Learner Settings → Appearance bo'limida: Light/Dark/System, oldindan belgilangan accent ranglar, dashboard wallpaper, wallpaper opacity/overlay, blur, sidebar ko'rinishi, compact/comfortable layout, motion preferences. Default holatda interfeys minimalistik va neytral. Moslashtirish kontentni o'qishga xalaqit bermasin.

### 4.5. Wallpaper boshqaruvi
Settings → Appearance → Wallpaper: standart galereyadan tanlash, custom yuklash, almashtirish, olib tashlash, opacity va blur, Light va Dark uchun alohida wallpaper. Wallpaper faqat dashboard kabi mos sahifalarga qo'llanadi va reading matnlari hamda exam interfeysining o'qilishiga xalaqit bermaydi. Yuklangan fayllar backendda tekshiriladi (size va format limitlari) va server yoki object storage'da saqlanadi. Mock exam vaqtida wallpaper va dekorativ effektlar o'chiriladi.

### 4.6. Preferences saqlanishi
Login qilgan learner uchun sozlamalar database'da saqlanadi va boshqa qurilmada ham tiklanadi. Login qilmagan foydalanuvchi uchun local preference ishlatiladi. O'zgarish darhol qo'llanadi.

## 5. Landing page
### 5.1. Header
Logo, How It Works, Features, Practice, Mock Exams, Pricing, FAQ, Sign In, Create Account. Mobile qurilmalarda responsive menu.

### 5.2. Hero
Platforma qiymatini tushuntiruvchi aniq headline, qisqa izoh, Start Preparing CTA, Explore Platform CTA, dashboard yoki mock exam preview. Marketing matnlari asossiz va'da bermasin (ball kafolatlanmaydi).

### 5.3. Bo'limlar
1. Hero
2. Asosiy imkoniyatlar
3. Daily Practice
4. Mock Exam
5. AI Study Assistant
6. Personalized Study Plan
7. Progress Analytics
8. How It Works
9. Pricing
10. FAQ
11. Footer (Privacy Policy, Terms of Service, Contact va boshqa havolalar)

Sahifa SEO talablariga mos bo'lsin.

## 6. Authentication
Learner va Owner uchun alohida flowlar.

### 6.1. Learner
**Sign In:** Email, Password, Remember Me (session modeliga qarab), Sign In, Forgot Password, Continue with Google, Create Account havolasi. Xatolar tushunarli ko'rsatiladi.

**Create Account:** Google orqali yoki email orqali. Email flow:
1. Email kiritiladi.
2. Verification code yuboriladi.
3. Kod tasdiqlanadi.
4. Account yaratiladi.
5. Profile Setup ochiladi.

Kodning amal qilish muddati, urinishlar limiti va qayta yuborish cheklovi bo'lsin.

**Google:** OAuth orqali kiriladi, alohida password talab qilinmaydi. Keyinroq boshqa login usuli qo'shilishi mumkin.

**Xavfsizlik:** passwordlar xavfsiz hashing bilan saqlanadi, password reset flow bor, session va tokenlar xavfsiz boshqariladi, login endpointlarida rate limiting bor.

### 6.2. Owner
Owner login alohida route'da: learner `/auth/sign-in`, owner `/owner/login`. Owner accountlari learner registration orqali yaratilmaydi, ular xavfsiz admin jarayoni orqali yaratiladi yoki seed qilinadi. Talablar: email va password, zarur bo'lsa MFA/2FA, secure session, login urinishlarini cheklash, owner route'larini backendda himoyalash, har bir muhim admin harakatini audit logga yozish. Owner sahifasini faqat frontendda yashirish xavfsizlik chorasi hisoblanmaydi.

## 7. Profile Setup
Email tasdiqlangach to'ldiriladigan maydonlar: Avatar, First Name, Last Name, Email, Phone Number (ixtiyoriy), Date of Birth yoki yosh ma'lumoti, Password va Password Confirmation (email/password account uchun). Email tasdiqlangan email bilan bir xil bo'ladi va faqat alohida verification flow orqali o'zgartiriladi. Standart avatar ham bor. Google foydalanuvchisining ma'lumotlari oldindan to'ldiriladi. Profile Setup tugagach onboarding boshlanadi.

## 8. Onboarding
Qisqa va bosqichma-bosqich.
- **8.1. Maqsad:** maqsadli SAT balli, imtihon sanasi, SAT topshirish sababi (ixtiyoriy).
- **8.2. Study preferences:** kunlik vaqt, haftadagi o'qish kunlari, Math va R&W bo'yicha o'ziga ishonch darajasi, oldingi SAT natijasi (bo'lsa).
- **8.3. Talablar:** progress indicator, Back va Continue, tanlovlarni keyin o'zgartirish, ixtiyoriy savollarni Skip qilish, har bir bosqichni saqlash. Oxirida Placement Assessment sahifasiga o'tiladi.

## 9. Placement test va darajani aniqlash
Learner uch yo'ldan birini tanlaydi:
1. Oldingi SAT natijasini kiritish yoki tayyorgarligini taxminiy belgilash.
2. Placement Test topshirish.
3. Testsiz asosiy mavzulardan boshlash.

### 9.1. Placement Test
Math va R&W bo'yicha diagnostik savollar. Natijada: umumiy diagnostik profil, Math va R&W bo'yicha kuchli va zaif domainlar, tavsiya etilgan boshlang'ich mavzular, study plan uchun boshlang'ich ma'lumot. Natija rasmiy SAT score sifatida ko'rsatilmaydi. Savollar banki va scoring modeli moslashuvchan bo'lsin.

### 9.2. Study Plan yaratish
Dastlabki reja quyidagilarga tayanadi: maqsadli ball, imtihongacha qolgan vaqt, kunlik o'qish vaqti, domain natijalari, mavjud progress. Dastlab rule-based algoritm ishlatiladi, AI tavsiyalari keyinroq qo'shiladi.

## 10. Learner Dashboard
### 10.1. Layout
Sidebar, Header, Main Content. Sidebar yig'iladi. Mobile qurilmalarda drawer yoki ixcham navigation.

### 10.2. Sidebar
- **Learn:** Dashboard, My Study Plan, Learn & Practice, Mock Exams, Formula & Concept Library, AI Study Assistant.
- **Track:** My Progress, Saved Questions, Mistake Notebook.
- **Account:** My Profile, Settings, Subscription.

### 10.3. Header
Page title yoki breadcrumb, Search (zarur bo'lsa), Notifications, streak yoki progress indikator, Avatar, Profile dropdown. Barcha learner sahifalarida minimalistik va izchil.

### 10.4. Dashboard Home
1. Welcome
2. Target SAT score
3. So'nggi diagnostik yoki mock natija
4. Today's Study Plan
5. Daily Practice CTA
6. Mock Exam CTA
7. Math va R&W progressi
8. Weak Topics
9. Continue Learning
10. Recent Activity
11. Study streak (faoliyat bo'lsa)

Ma'lumot bo'lmasa empty state ko'rsatiladi. Soxta ball yoki fake progress ko'rsatilmaydi.

## 11. My Study Plan
Daily va weekly view, vazifalar ro'yxati va davomiyligi, bajarilgan/bajarilmagan holat, vazifani boshlash va davom ettirish, rejani yangilash, kunlik vaqtni va imtihon sanasini o'zgartirish. Har bir vazifa lesson, practice set yoki review sessiyasiga bog'langan. Reja o'zgarsa kelajakdagi vazifalar qayta hisoblanadi, oldingi natijalar saqlanadi.

## 12. Learn & Practice
Curriculum rasmiy 8 ta content domain asosida tuziladi.
- **12.1. Math:** Algebra; Advanced Math; Problem-Solving and Data Analysis; Geometry and Trigonometry.
- **12.2. Reading & Writing:** Information and Ideas; Craft and Structure; Expression of Ideas; Standard English Conventions.

Har bir domain ichida topic va subtopiclar bor.

### 12.3. Lesson sahifasi
Title, learning objectives, explanation, formula va qoidalar, worked examples, practice questions, related topics, bookmark, lesson progress.

### 12.4. Practice flow
1. Learner topic tanlaydi.
2. Practice setni boshlaydi.
3. Savol va variantlar ko'rsatiladi.
4. Learner javob beradi.
5. Tizim javobni tekshiradi.
6. Explanation ko'rsatiladi.
7. Learner savolni saqlashi yoki qayta ishlashi mumkin.
8. Set oxirida natija ko'rsatiladi.

Practice Mode'da feedback darhol beriladi. Test Mode'da javoblar test tugaguncha yashiriladi.

### 12.5. Savol turlari
V1 multiple-choice'ni to'liq qo'llab-quvvatlaydi, Math uchun numeric answer ham bor. Ma'lumot modeli yangi savol turlarini qo'shishga tayyor bo'lsin.

## 13. Question Bank
Learner va Owner tizimlarining markaziy moduli. Har bir savolda: unique ID, section, domain, topic, difficulty, content, variantlar (kerak bo'lsa), answer key, explanation, source va license, status, version, created_at va updated_at. Filterlar: section, domain, topic, difficulty, question type, publication status. Published savollar versiyalanadi: tahrir oldingi attemptlarni buzmaydi. Faqat original yoki foydalanish huquqi bor savollar ishlatiladi.

## 14. Mock Exams
Imkon qadar real Digital SAT tajribasiga yaqin bo'lsin.
- **14.1. Turlar:** Full-Length, Math Practice Test, R&W Practice Test, Section-specific practice.
- **14.2. Rasmiy vaqtlar:** R&W 64 daqiqa, 54 savol, 2 modul. Math 70 daqiqa, 44 savol, 2 modul. Tanaffus 10 daqiqa. Jami 2 soat 14 daqiqa. Owner standart vaqtlarni o'zboshimchalik bilan o'zgartirmaydi. Practice formatlarning boshqa vaqtlari aniq ko'rsatiladi.
- **14.3. Instructions:** test nomi, format, savollar soni, vaqt, qoidalar, Start Exam. Timer learner tasdiqlagandan keyin boshlanadi.
- **14.4. Interface:** chalg'ituvchi elementlarsiz alohida layout (dashboard sidebar'i yo'q). Unda countdown timer, module nomi, savol, variantlar, savol navigatori, Mark for Review, Next/Previous, javob berilganini ko'rsatuvchi indikator, module'ni tugatish va tasdiqlash bor.
- **14.5. Reliability:** javoblar muntazam backendga saqlanadi, refresh'da yo'qolmaydi. Server-side timer va deadline ishlatiladi, timeout backendda tekshiriladi. Attempt ikki marta yakunlanmaydi, tarmoq uzilganda tiklanadi, submit idempotent.
- **14.6. Adaptive:** rasmiy algoritmni taqlid qilinmaydi. V1 fixed-form exam bo'ladi. Savollar kalibrlangach module routing va scoring alohida ishlab chiqiladi.

## 15. Exam Results va Review
Umumiy performance, Math va R&W natijalari, to'g'ri, noto'g'ri va javobsiz savollar, domain bo'yicha natijalar, vaqt sarfi, savolma-savol review, to'g'ri javoblar va explanationlar, tavsiya mavzular, Practice Again tugmasi. Scoring modeli validatsiya qilinmagan bo'lsa, rasmiy score o'rniga practice performance yoki estimated score range ko'rsatiladi.

## 16. My Progress
Practice activity, mock exam tarixi, Math va R&W progressi, domain mastery, accuracy, o'rtacha javob vaqti, haftalik o'qish davomiyligi, study plan completion, xatolar va takroriy xatolar soni, vaqt bo'yicha progress grafigi. Hammasi faqat real saqlangan ma'lumotlardan hisoblanadi.

## 17. Saved Questions va Mistake Notebook
**Saved Questions:** saqlanganlar ro'yxati, topic bo'yicha filter, savolni ochish, bookmarkni olib tashlash, qayta ishlash.

**Mistake Notebook:** noto'g'ri javob berilgan savollar avtomatik qo'shiladi. Har bir yozuvda: savol, learner javobi, to'g'ri javob, explanation, mavzu, xato sanasi, qayta urinishlar, "o'zlashtirdim" belgisi. Ma'lumot modeli spaced repetition'ga tayyor bo'lsin.

## 18. Formula va Concept Library
Har bir formulada: nom, formula, izoh, qayerda ishlatilishi, misol, tegishli mavzu, related practice questions. R&W uchun grammar va vocabulary tushunchalari ham bo'lishi mumkin. Search va bookmark bor.

## 19. AI Study Assistant
Savolni tushuntirish, yechimni bosqichma-bosqich ko'rsatish, formulani izohlash, xato javob sababini tushuntirish, o'xshash savol yaratish, mavzu bo'yicha qo'shimcha misol berish. AI imkon qadar platformadagi savol va lesson ma'lumotlaridan foydalanadi. Matematik yechimlar tekshiriladi, muhim kontentni Owner tasdiqlaydi. Usage limits va rate limits bor.

## 20. Learner Profile va Settings
- **Account:** profil, avatar, email, telefon, password va login usullari, account deletion.
- **Learning Preferences:** target score, exam date, daily study time, study days, learning preferences.
- **Appearance:** Light/Dark/System, accent, wallpaper gallery va upload, opacity, blur, sidebar preferences, compact/comfortable.
- **Notifications:** study reminders, plan reminders, weekly summary, email va in-app bildirishnomalar.
- **Subscription:** joriy plan, feature limits, billing details, payment history, cancel.

## 21. Owner Dashboard
Owner uchun alohida dizayn tizimi va navigation, learner bilan bir xil minimalistik tamoyillar. Owner Light/Dark/System, sidebar holati, compact layout va (ruxsat berilsa) accent'ni tanlaydi. Owner theme sozlamalari learner'nikidan mustaqil.
- **21.1. Owner Authentication:** alohida route, email va password, MFA, secure session, RBAC, login rate limiting, audit logging. Learner o'z accountidan Owner bo'la olmaydi.
- **21.2. Home ko'rsatkichlari:** total learners, new registrations, active learners, practice activity, completed mock exams, content inventory, published va draft content, subscription stats, so'nggi admin harakatlari. Demo data production statistikasi sifatida ko'rsatilmaydi.
- **21.3. Sidebar:**
  - Overview: Dashboard, Learners, Analytics.
  - Content Management: Curriculum CMS, Lessons, Question Bank, Mock Exam Builder, Formula Library, AI Content Assistant.
  - Platform: Plans & Subscriptions, Notifications, Platform Settings, Audit Logs.

## 22. Owner Content CMS
Bitta CMS orqali boshqariladi: curriculum domains, topics va subtopics, lessons, practice questions, formula library, mock exams, AI-generated content.
- **22.1. Status:** Draft, In Review, Published, Archived. Learnerlar faqat Published kontentni ko'radi.
- **22.2. Lesson Editor:** title, rich text yoki structured content, headings, lists, formulas, images, worked examples, related questions, preview, Save Draft, Publish.
- **22.3. Navigation:** bo'limlar accordion ko'rinishida bo'lishi mumkin. Label bosilganda ham ochilib-yopilsin, faqat arrow tugmasi orqali emas.

## 23. Owner AI Content Assistant
- **23.1. Imkoniyatlar:** lesson explanation, savol explanation, worked example, practice savollar taklifi, qo'shimcha misollar, kontentni soddalashtirish, grammatik va matematik tekshiruv, tarjima, SAT domain va topic'ga moslikni tekshirish.
- **23.2. Flow:**
  1. Domain tanlanadi.
  2. Topic tanlanadi.
  3. Create with AI bosiladi.
  4. Kontent turi tanlanadi.
  5. Talablar kiritiladi.
  6. AI draft yaratadi.
  7. Draft editorda ochiladi.
  8. Owner tahrirlaydi.
  9. Preview qiladi.
  10. Save Draft yoki Publish qiladi.

  AI hech qachon avtomatik publish qilmaydi.
- **23.3. Xavfsizlik va sifat:** AI javoblari strukturali formatda olinadi, savol va javoblar tekshiriladi, to'g'ri javob va explanation mosligi tekshiriladi, kerak joyda avtomatik mathematical validation ishlatiladi. Xatoda tushunarli xabar ko'rsatiladi. API kalitlari frontendga chiqmaydi. Rate limiting va cost control bor. AI kontenti va uning statusi saqlanadi.

## 24. Mock Exam Builder
Title va description, exam type, module configuration, question selection, domain va difficulty distribution, time configuration, preview, validation, Publish, Unpublish, Archive. Publish'dan oldin majburiy maydonlar tekshiriladi. Savollar soni yoki module konfiguratsiyasi noto'g'ri bo'lsa xato ko'rsatiladi. Published exam versiyalari saqlanadi.

## 25. Learner Management
Jadval: avatar, name, email, registration date, account status, subscription plan, last activity. Learner profilida: account ma'lumotlari, learning progress, practice activity, mock exam tarixi, subscription holati. Owner passwordni ko'ra olmaydi. Admin harakatlari audit logga yoziladi.

## 26. Analytics
Registrations over time, active learners, practice completion, mock exam completion, eng ko'p o'rganiladigan domainlar, kontentdan foydalanish, subscription metrics. Hammasi haqiqiy ma'lumotdan hisoblanadi. Metric definitionlar va vaqt oraliqlari izchil.

## 27. Monetization va Subscriptions
Free va Premium. Tizim quyidagilarni qamraydi: plan definitions, feature access, usage limits, subscription status, billing cycle, payment history, cancellation, renewal, payment failure handling. Haqiqiy payment provider keyinroq ulanishi mumkin, lekin arxitektura bunga tayyor bo'lsin. Premium access backendda tekshiriladi. Payment status frontend ma'lumotiga qarab belgilanmaydi.

## 28. Xavfsizlik
Authentication, authorization, RBAC, password hashing, secure session va token boshqaruvi, rate limiting, input validation, SQL injection himoyasi, XSS va CSRF xavflarini kamaytirish, CORS, file upload validation, environment variables, secret management, audit logs, account deletion, data privacy, database backups. Learner faqat o'z ma'lumotlarini boshqaradi. Owner endpointlari backendda himoyalanadi.

## 29. Performance va Responsive
Desktop, laptop, tablet va mobile qo'llab-quvvatlanadi. Desktopda to'liq sidebar, mobile'da ixcham navigation. Mock exam kichik ekranda ham ishlaydi, test oldidan tavsiya etilgan qurilma talablari ko'rsatilishi mumkin. Har bir asosiy modulda loading, error, empty va success state'lar bor.

## 30. Accessibility
Klaviatura navigatsiyasi, ko'rinadigan focus, to'g'ri label va form validation, rangdan tashqari indikatorlar, yetarli kontrast, screen reader semantikasi, reduce motion sozlamasi.

## 31. Notifications
In-app bildirishnomalar: study plan reminder, tugallanmagan vazifa, haftalik summary, security event, subscription holati. Foydalanuvchi sozlamalariga mos ishlaydi.

## 32. Error handling va Empty states
Har bir modulda loading, empty, error va success state'lar, kerak joyda retry action bor. Masalan, mock natijasi hali yo'q bo'lsa, bo'sh grafik o'rniga birinchi mock exam'ni boshlashga undovchi ekran ko'rsatiladi.

## 33. Testing
Loyiha testlarsiz yakunlangan hisoblanmaydi.
- **Frontend:** component, form validation, navigation va auth flow testlari.
- **Backend:** unit, API integration, auth va authorization, database integration, scoring testlari.
- **E2E:**
  1. Registration
  2. Email verification
  3. Login va logout
  4. Profile setup
  5. Onboarding
  6. Placement test
  7. Daily practice
  8. Mock exam
  9. Exam results
  10. Learner settings
  11. Owner login
  12. CMS content creation
  13. AI draft creation
  14. Content publishing
  15. Learner access restrictions

## 34. MVP bosqichlari
- **Phase 1, Foundation:** repo audit, project structure, design tokens, landing, learner va owner auth, profile setup, onboarding, database foundation, learner va owner layoutlari.
- **Phase 2, Core Learning:** curriculum, Question Bank CMS, Lesson CMS, Daily Practice, answer checking, explanations, Saved Questions, Mistake Notebook.
- **Phase 3, Mock Exams:** builder, exam interface, timer, answer persistence, submission, results, review mode.
- **Phase 4, Personalization:** placement test, study plan, progress analytics, AI Study Assistant, Owner AI Content Assistant.
- **Phase 5, Production Readiness:** subscription arxitekturasi, payment integration, security review, performance, accessibility, deployment, monitoring va backups.

## 35. Acceptance Criteria
Modul quyidagi shartlar bajarilganda tayyor hisoblanadi:
- UI talablarga mos.
- Responsive ishlaydi.
- Kerak bo'lsa backend integratsiyasi bor.
- Ma'lumotlar DB'da to'g'ri saqlanadi.
- Auth va authorization ishlaydi.
- Loading, error va empty state'lar bor.
- Asosiy flowlar test qilingan.
- Soxta muvaffaqiyat xabarlari yo'q.
- Console va backend loglarida hal qilinmagan kritik xato yo'q.
- Hujjatlar yangilangan.

## 36. Claude uchun ish qoidalari
1. Ishni boshlashdan oldin repository'ni tekshir.
2. Mavjud kod, konfiguratsiya va loyiha holatini aniqlab ol.
3. Hech qanday muhim faylni tekshirmasdan o'chirma.
4. PRD'ni texnik modullarga ajrat.
5. Avval architecture va implementation plan taqdim et.
6. Muhim texnik qarorlarni qisqacha asosla.
7. Barcha funksiyalarni bir vaqtda yozishga urinma.
8. Har bir bosqichni alohida ishlab chiq va test qil.
9. Frontend va backend kontraktlarini mos saqla.
10. API endpointlarini hujjatlashtir.
11. DB o'zgarishlari uchun migrationlardan foydalan.
12. Hardcoded demo ma'lumotlarni real production ma'lumoti sifatida ko'rsatma.
13. API kalitlari va secretlarni kodga yozma.
14. Keraksiz dependency qo'shma.
15. UI komponentlarini qayta ishlat.
16. Barcha asosiy sahifalarda yagona design system ishlat.
17. Har bir muhim funksiya uchun acceptance criteria'ni tekshir.
18. Tugallanmagan funksiyalarni tayyor deb e'lon qilma.
19. Har bir bosqich oxirida bajarilgan ishlar, testlar, muammolar va keyingi qadamlarni yoz.
20. PRD'da aniqlashtirilmagan, xavfsizlik yoki arxitekturaga ta'sir qiluvchi qarorlar bo'lsa, taxmin qilish o'rniga savol ber.

## 37. Birinchi topshiriq (0.00-Foundation sessiyasi uchun)
Kod yozishni boshlama. Avval:
1. Repository'ni tekshir.
2. Mavjud texnologiyalar va fayl strukturasini tahlil qil.
3. PRD'ni texnik modullarga ajrat.
4. Frontend va backend arxitekturasini taklif qil.
5. DB entity va relationship'larini rejalashtir.
6. API modullarini rejalashtir.
7. Auth va authorization modelini tushuntir.
8. Design system va sahifalar ro'yxatini tuz.
9. MVP'ni kichik, tekshiriladigan bosqichlarga ajrat.
10. Xavfli yoki aniqlashtirilishi kerak bo'lgan qarorlarni sanab chiq.

Natijani ko'rib chiqish uchun taqdim et. Tasdiqlanmaguncha katta kod o'zgarishlarini boshlama.

Asosiy maqsad: ishonchli, kengaytiriladigan, xavfsiz va professional SAT self-study platforma.
