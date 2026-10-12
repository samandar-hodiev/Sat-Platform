import type { NextConfig } from "next";

// Headers that cannot break the app. A script Content-Security-Policy needs nonces and is
// left to the security review (5.02).
const securityHeaders = [
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "Permissions-Policy", value: "camera=(), microphone=(), geolocation=()" },
  {
    key: "Content-Security-Policy",
    value: "frame-ancestors 'none'; object-src 'none'; base-uri 'self'; form-action 'self'",
  },
];

const nextConfig: NextConfig = {
  reactStrictMode: true,
  poweredByHeader: false,
  // Development-only pages (/dev/*, e.g. the UI kit) do not exist in production. A rewrite is
  // used instead of notFound(): the root loading.tsx starts streaming with status 200 first.
  async rewrites() {
    return {
      beforeFiles:
        process.env.NODE_ENV === "production"
          ? [{ source: "/dev/:path*", destination: "/_dev-disabled" }]
          : [],
      afterFiles: [],
      fallback: [],
    };
  },
  async headers() {
    return [{ source: "/:path*", headers: securityHeaders }];
  },
};

export default nextConfig;
