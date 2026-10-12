// Learner profile (session 2.01). Contract: docs/API.md → Profile.
export {
  avatarSrc,
  displayName,
  fetchProfile,
  type LearnerProfile,
  type ProfileSetupInput,
} from "./api";
export { ProfileGate } from "./components/ProfileGate";
export { ProfileSetup } from "./components/ProfileSetup";
export { setProfile, useLearnerProfile, type ProfileState } from "./profileStore";
export { AFTER_PROFILE_SETUP, PROFILE_SETUP_ROUTE } from "./routes";
