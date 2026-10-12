/** Joins class names, skipping falsy values. Components put the caller's className last. */
export function cn(...classes: Array<string | false | null | undefined>): string {
  return classes.filter(Boolean).join(" ");
}
