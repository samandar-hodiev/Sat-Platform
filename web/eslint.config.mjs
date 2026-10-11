import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

export default defineConfig([
  ...nextVitals,
  ...nextTs,
  {
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/consistent-type-imports": "warn",
      // Tokens and personal data never go to localStorage (readable by any injected script).
      "no-restricted-globals": [
        "error",
        {
          name: "localStorage",
          message: "Never store tokens or user data in localStorage; see docs/AUTHENTICATION.md.",
        },
      ],
    },
  },
  globalIgnores([".next/**", "out/**", "build/**", "coverage/**", "next-env.d.ts"]),
]);
