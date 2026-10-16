// Package preferences is the learner's account-wide appearance settings (session 2.11, PRD
// §4.4–4.6, §20 Appearance): theme, accent, density, motion, sidebar state and the dashboard
// wallpaper (gallery pattern or uploaded image, per colour mode, with opacity and blur). It owns
// the user_preferences table. The web keeps a first-party cookie as the device copy and syncs
// it with GET/PATCH /me/preferences; the account wins once something was stored.
package preferences
