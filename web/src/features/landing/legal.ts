// Privacy Policy and Terms of Service (session 0.03). These are TEMPLATES describing how the
// platform is built (see docs/PRD.md); they have not been reviewed by a lawyer. Before launch:
// fill LEGAL_CONFIG, have both documents reviewed, then set `reviewed: true` to remove the
// draft notice shown on the pages.

import { SITE_NAME } from "@/lib/site";

export const LEGAL_CONFIG = {
  /** Legal name of the company or person operating the platform. */
  operator: SITE_NAME,
  /** "Last updated" date shown on both pages (ISO yyyy-mm-dd), set when the text is final. */
  lastUpdated: null as string | null,
  /** Country whose law governs the Terms; filled in during legal review. */
  governingLaw: null as string | null,
  /** True once a lawyer has reviewed the texts; hides the draft notice. */
  reviewed: false,
};

export interface LegalSection {
  id: string;
  title: string;
  paragraphs?: string[];
  list?: string[];
}

export interface LegalDocument {
  title: string;
  summary: string;
  sections: LegalSection[];
}

const operator = LEGAL_CONFIG.operator;

export const PRIVACY_POLICY: LegalDocument = {
  title: "Privacy Policy",
  summary: `How ${operator} collects, uses and protects your personal data.`,
  sections: [
    {
      id: "who-we-are",
      title: "Who we are",
      paragraphs: [
        `${operator} ("we", "us") operates a self-study platform for the Digital SAT. This policy ` +
          "explains what personal data we process when you visit the site or use your account, " +
          "why we process it and what choices you have.",
      ],
    },
    {
      id: "data-we-collect",
      title: "Data we collect",
      list: [
        "Account data: your name, email address, password (stored only as a secure hash) or the " +
          "Google account details you choose to share when signing in with Google.",
        "Profile and goals: an optional avatar, target score, exam date, daily study time and " +
          "similar learning preferences you enter.",
        "Learning activity: placement test answers, practice and mock exam answers, timing, " +
          "results, bookmarks, mistake notebook entries and lesson progress.",
        "AI assistant usage: the questions you ask the assistant and its answers, used to " +
          "provide the feature and to enforce usage limits.",
        "Appearance settings: theme, accent and layout choices, stored in your account and in " +
          "a first-party cookie.",
        "Technical data: IP address, browser type and request logs needed to secure and " +
          "operate the service.",
        "Subscription data: plan, status and payment history. Card details are handled by " +
          "the payment provider, not stored by us.",
      ],
    },
    {
      id: "how-we-use",
      title: "How we use your data",
      list: [
        "To create and secure your account and keep you signed in.",
        "To build your study plan and show your progress, practice performance and estimated " +
          "score ranges.",
        "To provide the AI study assistant within its usage limits.",
        "To send account emails such as verification codes, password resets and, if you allow " +
          "them, study notifications.",
        "To prevent abuse, investigate problems and improve the platform using aggregated " +
          "statistics.",
      ],
    },
    {
      id: "sharing",
      title: "Sharing with service providers",
      paragraphs: [
        "We do not sell your personal data. We share it only with providers that help us run " +
          "the platform, such as hosting, email delivery, the AI model provider and the payment " +
          "provider, and only as much as each needs to perform its service.",
      ],
    },
    {
      id: "cookies",
      title: "Cookies",
      paragraphs: [
        "We use first-party cookies that are needed for the site to work: to keep you signed " +
          "in and to remember your appearance settings. We do not use advertising cookies.",
      ],
    },
    {
      id: "retention",
      title: "Data retention and deletion",
      paragraphs: [
        "We keep your data while your account is active. You can delete your account from your " +
          "profile settings; your personal data is then deleted or anonymized, except records " +
          "we must keep for legal, security or accounting reasons.",
      ],
    },
    {
      id: "your-rights",
      title: "Your rights",
      paragraphs: [
        "Depending on where you live, you may have the right to access, correct, export or " +
          "delete your personal data and to object to some processing. Use the Contact page to " +
          "make a request.",
      ],
    },
    {
      id: "security",
      title: "Security",
      paragraphs: [
        "We protect data with encrypted connections, hashed passwords, access controls and " +
          "audit logs for administrative actions. No system is perfectly secure, so please use " +
          "a strong, unique password.",
      ],
    },
    {
      id: "children",
      title: "Children",
      paragraphs: [
        "The platform is intended for students preparing for the SAT. If you are under the age " +
          "of digital consent in your country, use the platform with the permission of a parent " +
          "or guardian.",
      ],
    },
    {
      id: "changes",
      title: "Changes to this policy",
      paragraphs: [
        "We will update this page when the policy changes and, for significant changes, notify " +
          "signed-in users.",
      ],
    },
  ],
};

export const TERMS_OF_SERVICE: LegalDocument = {
  title: "Terms of Service",
  summary: `The rules for using ${operator}.`,
  sections: [
    {
      id: "acceptance",
      title: "Acceptance of these terms",
      paragraphs: [
        `By creating an account or using ${operator} you agree to these Terms and to the ` +
          "Privacy Policy. If you do not agree, do not use the platform.",
      ],
    },
    {
      id: "service",
      title: "The service",
      paragraphs: [
        `${operator} is an independent self-study tool for the Digital SAT. It is not ` +
          "affiliated with or endorsed by the College Board. It does not register you for the " +
          "SAT and does not issue official scores.",
      ],
    },
    {
      id: "no-guarantee",
      title: "No score guarantee",
      paragraphs: [
        "Practice results, mock exam results and score ranges on the platform are estimates " +
          "for practice only. We do not guarantee any result on the official SAT.",
      ],
    },
    {
      id: "account",
      title: "Your account",
      list: [
        "Provide accurate information and keep your password secure.",
        "Your account is personal; do not share it or let others use it.",
        "Tell us promptly if you believe your account has been accessed without permission.",
      ],
    },
    {
      id: "acceptable-use",
      title: "Acceptable use",
      list: [
        "Do not copy, scrape, resell or publish platform content such as questions, lessons " +
          "or explanations.",
        "Do not try to bypass usage limits, exam timers, plan restrictions or security measures.",
        "Do not use the AI assistant to produce harmful or unlawful content.",
        "Do not interfere with the platform or other users.",
      ],
    },
    {
      id: "plans",
      title: "Plans and payments",
      paragraphs: [
        "The platform offers a Free plan and a Premium plan. Premium prices, billing cycles, " +
          "renewal and cancellation terms are shown before you subscribe. Premium access " +
          "depends on a confirmed payment.",
      ],
    },
    {
      id: "content",
      title: "Intellectual property",
      paragraphs: [
        `Platform content and software belong to ${operator} or its licensors. You may use ` +
          "them only for your personal SAT preparation.",
      ],
    },
    {
      id: "ai",
      title: "AI-generated explanations",
      paragraphs: [
        "AI assistant answers are generated automatically and may contain mistakes. Check " +
          "important explanations against lessons and official materials.",
      ],
    },
    {
      id: "termination",
      title: "Suspension and termination",
      paragraphs: [
        "You can stop using the platform and delete your account at any time. We may suspend " +
          "accounts that break these Terms.",
      ],
    },
    {
      id: "liability",
      title: "Disclaimers and liability",
      paragraphs: [
        'The platform is provided "as is". To the extent permitted by law, we are not liable ' +
          "for indirect losses or for results on any official exam.",
      ],
    },
    {
      id: "changes",
      title: "Changes to these terms",
      paragraphs: [
        "We may update these Terms. We will post the new version here and, for significant " +
          "changes, notify signed-in users before they take effect.",
      ],
    },
  ],
};
