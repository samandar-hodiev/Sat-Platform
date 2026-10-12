// Package profile is the learner's personal details (session 2.01, PRD §7): Profile Setup after
// the email is verified, the avatar upload, and the read endpoint other screens use for the
// learner's name and picture (dashboard welcome, shell header).
//
// Profile rows live in the profiles table (0.01). Account data (email, password, linked sign-in
// methods) belongs to the auth module and is reached only through the Accounts interface, never
// by reading auth tables.
//
// Profile Setup is a one-time step: PUT /me/profile/setup fills the profile and marks it
// complete; onboarding (2.02) starts after it. Later edits are My Profile (2.10).
package profile
