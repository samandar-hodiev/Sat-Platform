// Package onboarding is the short, step-by-step questionnaire after Profile Setup (session 2.02,
// PRD §8): the learner's goal (target score, exam date, reason), study schedule (daily time,
// study days) and background (confidence in Math and Reading & Writing, an earlier SAT result).
//
// Each step is saved on its own (PUT /me/onboarding/<step>) so onboarding resumes where the
// learner stopped, and stays editable after onboarding is finished (Learning Preferences, 2.11).
// POST /me/onboarding/complete ends it; the learner then goes to Placement (3.11, PRD §9).
//
// Answers live in learner_study_profiles. The study plan (2.05) and the dashboard goal block
// (2.04) read them through Service.Get. The profile module is reached only through the Profiles
// interface (Profile Setup must be finished; the learner's time zone decides "today").
package onboarding
