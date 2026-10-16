// @vitest-environment jsdom
import { describe, expect, it } from "vitest";

import { readSidebarState, subscribeSidebarState, writeSidebarState } from "./sidebarState";

describe("sidebar state store (2.11)", () => {
  it("reads nothing before a choice is stored, then notifies subscribers of writes", () => {
    expect(readSidebarState()).toBeNull();
    const seen: (string | null)[] = [];
    const unsubscribe = subscribeSidebarState(() => seen.push(readSidebarState()));
    writeSidebarState("collapsed");
    writeSidebarState("expanded");
    unsubscribe();
    writeSidebarState("collapsed");
    expect(seen).toEqual(["collapsed", "expanded"]);
    expect(readSidebarState()).toBe("collapsed");
  });
});
