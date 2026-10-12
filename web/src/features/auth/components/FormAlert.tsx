import type { ReactNode } from "react";

import { AlertCircleIcon, CheckCircleIcon } from "@/components/ui";
import { cn } from "@/lib/cn";

/** Form-level message above the fields. Errors are announced by screen readers (role="alert"). */
export function FormAlert({
  variant = "error",
  children,
}: {
  variant?: "error" | "success";
  children: ReactNode;
}) {
  return (
    <div
      role={variant === "error" ? "alert" : "status"}
      className={cn(
        "flex items-start gap-2.5 rounded-md px-3 py-2.5 text-sm",
        variant === "error" ? "bg-error-soft text-error-text" : "bg-success-soft text-success-text",
      )}
    >
      <span className="mt-0.5 shrink-0">
        {variant === "error" ? <AlertCircleIcon size={16} /> : <CheckCircleIcon size={16} />}
      </span>
      <span>{children}</span>
    </div>
  );
}
