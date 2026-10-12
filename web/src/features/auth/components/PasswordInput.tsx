"use client";

import { useState } from "react";

import { Input, type InputProps } from "@/components/ui/Input";

import { EyeIcon, EyeOffIcon } from "./icons";

/** Input with a show/hide toggle. Use inside FormField so it gets its label and error wiring. */
export function PasswordInput({ className, ...props }: Omit<InputProps, "type">) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="relative">
      <Input
        {...props}
        type={visible ? "text" : "password"}
        className={`pr-10 ${className ?? ""}`}
      />
      <button
        type="button"
        onClick={() => setVisible((v) => !v)}
        aria-label={visible ? "Hide password" : "Show password"}
        aria-pressed={visible}
        className="focus-ring absolute inset-y-0 right-0 flex w-10 items-center justify-center rounded-r-md text-fg-subtle transition-colors duration-(--duration-fast) hover:text-fg"
      >
        {visible ? <EyeOffIcon size={16} /> : <EyeIcon size={16} />}
      </button>
    </div>
  );
}
