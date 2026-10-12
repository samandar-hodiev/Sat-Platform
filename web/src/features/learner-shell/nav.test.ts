import { describe, expect, it } from "vitest";

import { LEARNER_HOME, LEARNER_NAV, activeNavItem, findNavItem } from "./nav";

describe("learner navigation", () => {
  it("has the PRD §10.2 groups and items in order", () => {
    expect(
      LEARNER_NAV.map((group) => [group.label, group.items.map((item) => item.label)]),
    ).toEqual([
      [
        "Learn",
        [
          "Dashboard",
          "My Study Plan",
          "Learn & Practice",
          "Mock Exams",
          "Formula & Concept Library",
          "AI Study Assistant",
        ],
      ],
      ["Track", ["My Progress", "Saved Questions", "Mistake Notebook"]],
      ["Account", ["My Profile", "Settings", "Subscription"]],
    ]);
  });

  it("keeps every link unique and inside the learner app", () => {
    const hrefs = LEARNER_NAV.flatMap((group) => group.items.map((item) => item.href));
    expect(new Set(hrefs).size).toBe(hrefs.length);
    for (const href of hrefs) expect(href === LEARNER_HOME || href.startsWith("/app/")).toBe(true);
  });

  it.each([
    ["/app", "Dashboard"],
    ["/app/", "Dashboard"],
    ["/app/exams", "Mock Exams"],
    ["/app/exams/", "Mock Exams"],
    ["/app/exams/abc/results", "Mock Exams"],
    ["/app/settings/appearance", "Settings"],
  ])("marks %s as %s", (path, label) => {
    expect(activeNavItem(path)?.label).toBe(label);
  });

  it.each(["/app/unknown", "/application", "/apps", "/owner", "/"])(
    "does not let Dashboard claim %s",
    (path) => {
      expect(activeNavItem(path)).toBeUndefined();
    },
  );

  it("finds items by exact href only", () => {
    expect(findNavItem("/app/profile")?.label).toBe("My Profile");
    expect(findNavItem("/app/profile/edit")).toBeUndefined();
  });
});
