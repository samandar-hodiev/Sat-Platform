import { NextRequest } from "next/server";
import { describe, expect, it } from "vitest";

import { config, proxy } from "./proxy";

describe("proxy (route guard)", () => {
  it("sends visitors without a session to sign-in and back afterwards", () => {
    const res = proxy(new NextRequest("http://web.test/app/practice?set=3"));
    expect(res.status).toBe(307);
    const location = new URL(res.headers.get("location")!);
    expect(location.pathname).toBe("/auth/sign-in");
    expect(location.searchParams.get("next")).toBe("/app/practice?set=3");
  });

  it("lets signed-in visitors through", () => {
    const req = new NextRequest("http://web.test/app", { headers: { cookie: "sat_signed_in=1" } });
    const res = proxy(req);
    expect(res.headers.get("location")).toBeNull();
    expect(res.headers.get("x-middleware-next")).toBe("1");
  });

  it("guards the learner app, the exam runner and the setup steps only", () => {
    expect(config.matcher).toEqual([
      "/app",
      "/app/:path*",
      "/exam",
      "/exam/:path*",
      "/setup",
      "/setup/:path*",
    ]);
  });
});
