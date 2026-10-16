// Public site copy (session 0.03, PRD §5). Kept in one place so wording can be reviewed and,
// later, translated (PRD: en/uz/ru-ready). Rules for every string here:
// - no score guarantees or promised point gains (PRD §5.2);
// - no invented numbers: no learner counts, ratings, testimonials or prices;
// - the platform gives practice performance and estimated ranges, never an official SAT score.

import { AUTH_ROUTES } from "@/features/auth/routes";

export const PUBLIC_ROUTES = {
  home: "/",
  pricing: "/pricing",
  faq: "/faq",
  privacy: "/privacy",
  terms: "/terms",
  contact: "/contact",
  signIn: AUTH_ROUTES.signIn,
  signUp: AUTH_ROUTES.signUp,
} as const;

/** Section anchors on the landing page; header links point at them. */
export const SECTION_IDS = {
  features: "features",
  practice: "practice",
  mockExams: "mock-exams",
  assistant: "ai-assistant",
  studyPlan: "study-plan",
  progress: "progress",
  howItWorks: "how-it-works",
  pricing: "pricing",
  faq: "faq",
} as const;

export interface NavLink {
  label: string;
  href: string;
}

/** Header navigation, PRD §5.1 order. */
export const HEADER_NAV: NavLink[] = [
  { label: "How It Works", href: `/#${SECTION_IDS.howItWorks}` },
  { label: "Features", href: `/#${SECTION_IDS.features}` },
  { label: "Practice", href: `/#${SECTION_IDS.practice}` },
  { label: "Mock Exams", href: `/#${SECTION_IDS.mockExams}` },
  { label: "Pricing", href: PUBLIC_ROUTES.pricing },
  { label: "FAQ", href: PUBLIC_ROUTES.faq },
];

export const FOOTER_GROUPS: Array<{ title: string; links: NavLink[] }> = [
  {
    title: "Platform",
    links: [
      { label: "Features", href: `/#${SECTION_IDS.features}` },
      { label: "Daily Practice", href: `/#${SECTION_IDS.practice}` },
      { label: "Mock Exams", href: `/#${SECTION_IDS.mockExams}` },
      { label: "AI Study Assistant", href: `/#${SECTION_IDS.assistant}` },
      { label: "Study Plan", href: `/#${SECTION_IDS.studyPlan}` },
    ],
  },
  {
    title: "Get started",
    links: [
      { label: "Create Account", href: PUBLIC_ROUTES.signUp },
      { label: "Sign In", href: PUBLIC_ROUTES.signIn },
      { label: "How It Works", href: `/#${SECTION_IDS.howItWorks}` },
      { label: "Pricing", href: PUBLIC_ROUTES.pricing },
    ],
  },
  {
    title: "Support",
    links: [
      { label: "FAQ", href: PUBLIC_ROUTES.faq },
      { label: "Contact", href: PUBLIC_ROUTES.contact },
      { label: "Privacy Policy", href: PUBLIC_ROUTES.privacy },
      { label: "Terms of Service", href: PUBLIC_ROUTES.terms },
    ],
  },
];

export const TRADEMARK_NOTICE =
  "SAT® is a trademark registered by the College Board, which is not affiliated with, and does " +
  "not endorse, this platform. Practice results are estimates, not official SAT scores.";

export const HERO = {
  eyebrow: "Digital SAT self-study",
  title: "Prepare for the Digital SAT with a plan built around you",
  description:
    "Find your starting level, follow a daily study plan, practice Math and Reading & Writing, " +
    "learn from your mistakes and take timed mock exams in the real digital format.",
  primaryCta: { label: "Start Preparing", href: PUBLIC_ROUTES.signUp },
  secondaryCta: { label: "Explore Platform", href: `/#${SECTION_IDS.features}` },
  note: "Free to start. Results are practice estimates, not official SAT scores.",
} as const;

export type FeatureIcon =
  | "target"
  | "calendar"
  | "book"
  | "pencil"
  | "timer"
  | "notebook"
  | "chart"
  | "sparkles"
  | "formula";

export interface Feature {
  icon: FeatureIcon;
  title: string;
  description: string;
}

/** "Asosiy imkoniyatlar" — the whole learning loop at a glance (PRD §1). */
export const FEATURES: Feature[] = [
  {
    icon: "target",
    title: "Placement test",
    description:
      "A diagnostic test shows your current level in each SAT domain before you start studying.",
  },
  {
    icon: "calendar",
    title: "Personal study plan",
    description:
      "Your target score, exam date and daily time turn into a day-by-day plan you can follow.",
  },
  {
    icon: "book",
    title: "Lessons by domain",
    description: "Math and Reading & Writing lessons organized by SAT domain, topic and subtopic.",
  },
  {
    icon: "pencil",
    title: "Daily practice",
    description: "Focused question sets with instant checking and a clear explanation for each.",
  },
  {
    icon: "notebook",
    title: "Mistake notebook",
    description:
      "Every wrong answer is collected so you can review it, understand it and try it again.",
  },
  {
    icon: "timer",
    title: "Timed mock exams",
    description: "Full-length practice exams in the digital format with a real-time timer.",
  },
  {
    icon: "chart",
    title: "Progress tracking",
    description: "See how your accuracy and practice performance change across every domain.",
  },
  {
    icon: "sparkles",
    title: "AI study assistant",
    description: "Ask about a concept or a question you got wrong and get a step-by-step answer.",
  },
  {
    icon: "formula",
    title: "Formula & concept library",
    description: "Formulas, grammar rules and key concepts in one searchable place.",
  },
];

export interface Spotlight {
  id: string;
  eyebrow: string;
  title: string;
  description: string;
  points: string[];
  /** Kind of schematic illustration drawn next to the text. */
  visual: "practice" | "exam" | "assistant" | "plan" | "progress";
}

/** Sections 3–7 of PRD §5.3. */
export const SPOTLIGHTS: Spotlight[] = [
  {
    id: SECTION_IDS.practice,
    eyebrow: "Daily Practice",
    title: "Practice a little every day, with feedback on every question",
    description:
      "Choose a domain or follow today's plan. Work in Practice mode to check each answer as " +
      "you go, or in Test mode to check them all at the end.",
    points: [
      "Multiple-choice and student-produced response questions",
      "An explanation for every question, right or wrong",
      "Save questions to revisit them later",
    ],
    visual: "practice",
  },
  {
    id: SECTION_IDS.mockExams,
    eyebrow: "Mock Exams",
    title: "Rehearse exam day with full-length timed mock exams",
    description:
      "Mock exams follow the Digital SAT structure, with module timing kept on the server. " +
      "Your answers are saved as you go, so a lost connection does not lose your work.",
    points: [
      "Reading & Writing and Math sections with timed modules",
      "Results with a domain-by-domain breakdown",
      "Question-by-question review with explanations",
    ],
    visual: "exam",
  },
  {
    id: SECTION_IDS.assistant,
    eyebrow: "AI Study Assistant",
    title: "Get unstuck with an assistant that explains, not just answers",
    description:
      "When a concept or a question does not make sense, ask the assistant. It walks through " +
      "the reasoning step by step so you can solve the next one yourself.",
    points: [
      "Explanations for questions you answered incorrectly",
      "Help with concepts from your lessons",
      "Usage limits keep the assistant fair for everyone",
    ],
    visual: "assistant",
  },
  {
    id: SECTION_IDS.studyPlan,
    eyebrow: "Personalized Study Plan",
    title: "Know exactly what to study today",
    description:
      "Your plan is built from your placement results, target score, exam date and the time " +
      "you have each day. It adjusts as you practice and as your weak areas change.",
    points: [
      "Daily and weekly views of your tasks",
      "More time on the domains where you need it most",
      "Recalculate the plan when your schedule changes",
    ],
    visual: "plan",
  },
  {
    id: SECTION_IDS.progress,
    eyebrow: "Progress Analytics",
    title: "See where you are improving and where to focus next",
    description:
      "Track accuracy by domain, completed practice and mock exam results over time. Every " +
      "number comes from your own work on the platform.",
    points: [
      "Accuracy and activity by SAT domain",
      "Mock exam history with estimated score ranges",
      "Strong and weak areas highlighted",
    ],
    visual: "progress",
  },
];

export const HOW_IT_WORKS: Array<{ title: string; description: string }> = [
  {
    title: "Create your account",
    description: "Sign up with email or Google and set up your profile.",
  },
  {
    title: "Set your goal",
    description: "Tell us your target score, exam date and how much time you have each day.",
  },
  {
    title: "Take the placement test",
    description: "A diagnostic test finds your starting level in each domain.",
  },
  {
    title: "Follow your plan",
    description: "Study lessons, practice daily and review your mistakes as the plan suggests.",
  },
  {
    title: "Test yourself",
    description: "Take timed mock exams and track your progress until exam day.",
  },
];

export interface Plan {
  id: "free" | "premium";
  name: string;
  summary: string;
  highlights: string[];
  cta: NavLink;
  featured?: boolean;
}

/**
 * Plan overview for the public site. Prices, limits and the exact Free/Premium feature split
 * are defined by Subscription & Plans (session 3.13); until that API exists the site shows no
 * prices and no per-plan limits, only what each plan is for (PRICING_STATUS explains why).
 */
export const PLANS: Plan[] = [
  {
    id: "free",
    name: "Free",
    summary: "Start preparing at no cost and see how the platform works for you.",
    highlights: [
      "Create an account and set your goal",
      "Get started with lessons and practice",
      "Track your activity and progress",
    ],
    cta: { label: "Create a free account", href: PUBLIC_ROUTES.signUp },
  },
  {
    id: "premium",
    name: "Premium",
    summary: "Full access for learners preparing seriously for exam day.",
    highlights: [
      "Everything in Free",
      "Expanded access to practice, mock exams and the AI assistant",
      "Personal study plan and detailed analytics",
    ],
    cta: { label: "Start with Free", href: PUBLIC_ROUTES.signUp },
    featured: true,
  },
];

export const PRICING_STATUS = {
  price: "Pricing coming soon",
  note:
    "Prices and the exact limits of each plan will be published here before Premium becomes " +
    "available. Premium access is always confirmed by your account, never by this page.",
} as const;

export interface FaqItem {
  id: string;
  question: string;
  answer: string;
}

export const FAQ: FaqItem[] = [
  {
    id: "what-is",
    question: "What is SAT Platform?",
    answer:
      "A self-study platform for the Digital SAT. It combines a placement test, a personal " +
      "study plan, lessons, daily practice, a mistake notebook, timed mock exams, progress " +
      "tracking and an AI study assistant in one place.",
  },
  {
    id: "official",
    question: "Is this the official SAT or affiliated with the College Board?",
    answer:
      "No. SAT Platform is an independent preparation tool and is not affiliated with or " +
      "endorsed by the College Board. It does not register you for the SAT or issue scores.",
  },
  {
    id: "scores",
    question: "Will I get an official SAT score here?",
    answer:
      "No. Mock exams and practice show your practice performance and an estimated score " +
      "range. Only the official SAT gives an official score.",
  },
  {
    id: "guarantee",
    question: "Do you guarantee a score increase?",
    answer:
      "No one can honestly guarantee a score. The platform helps you find weak areas, " +
      "practice consistently and rehearse the exam format; your results depend on your own " +
      "preparation.",
  },
  {
    id: "format",
    question: "Do mock exams match the Digital SAT format?",
    answer:
      "Mock exams follow the Digital SAT structure: a Reading & Writing section and a Math " +
      "section, each split into timed modules. The timer runs on the server and your answers " +
      "are saved as you go.",
  },
  {
    id: "placement",
    question: "Why should I take the placement test first?",
    answer:
      "The placement test shows your current level in each SAT domain. Your study plan uses it " +
      "to spend more time where you need it most.",
  },
  {
    id: "ai",
    question: "How does the AI study assistant work?",
    answer:
      "You can ask it to explain a concept or a question you got wrong. It explains the " +
      "reasoning step by step. Usage limits apply so the assistant stays available for everyone.",
  },
  {
    id: "price",
    question: "How much does it cost?",
    answer:
      "You can start for free. Premium prices and plan limits will be published on the " +
      "Pricing page before Premium becomes available.",
  },
  {
    id: "devices",
    question: "Which devices can I use?",
    answer:
      "The platform works in modern browsers on desktop, tablet and phone. For mock exams we " +
      "recommend a computer or tablet, as on exam day.",
  },
  {
    id: "data",
    question: "How is my data used?",
    answer:
      "Your answers and activity are used to build your study plan and show your progress. " +
      "See the Privacy Policy for details, including how to delete your account.",
  },
];
