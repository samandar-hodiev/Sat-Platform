// Web session endpoints (session 1.01): sign-in, sign-up/verify, google, refresh, logout.
// Logic and security notes: src/features/auth/server/sessionHandler.ts.

import { createSessionHandler } from "@/features/auth/server/sessionHandler";
import { apiUrl } from "@/lib/env";

export const dynamic = "force-dynamic";

let handler: ReturnType<typeof createSessionHandler> | undefined;

export async function POST(request: Request, ctx: { params: Promise<{ action: string[] }> }) {
  handler ??= createSessionHandler({
    apiOrigin: process.env.API_INTERNAL_URL?.replace(/\/+$/, "") || apiUrl(),
    secureCookies: process.env.NODE_ENV === "production",
  });
  const { action } = await ctx.params;
  return handler(request, action.join("/"));
}
