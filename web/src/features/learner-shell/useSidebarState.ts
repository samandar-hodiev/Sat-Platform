"use client";

import { useSyncExternalStore } from "react";

import { readSidebarState, subscribeSidebarState, type SidebarState } from "./sidebarState";

/**
 * The stored sidebar state. `serverValue` is what the server read from the request cookie: it is
 * used during hydration and while nothing is stored on this device.
 */
export function useSidebarState(serverValue: SidebarState): SidebarState {
  return useSyncExternalStore(
    subscribeSidebarState,
    () => readSidebarState() ?? serverValue,
    () => serverValue,
  );
}
