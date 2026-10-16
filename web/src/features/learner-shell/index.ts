// Learner layout (session 2.03). Rules: docs/ARCHITECTURE.md → Learner shell.
export { useShellAccount, type ShellAccount, type ShellUser } from "./account";
export { LearnerShell } from "./components/LearnerShell";
export {
  LEARNER_HOME,
  LEARNER_NAV,
  activeNavItem,
  findNavItem,
  type NavGroup,
  type NavItem,
} from "./nav";
export {
  SIDEBAR_COOKIE,
  parseSidebarState,
  writeSidebarState,
  type SidebarState,
} from "./sidebarState";
export { useSidebarState } from "./useSidebarState";
