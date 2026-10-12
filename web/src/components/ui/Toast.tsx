"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

import { IconButton } from "./IconButton";
import { AlertCircleIcon, AlertTriangleIcon, CheckCircleIcon, InfoIcon, XIcon } from "./icons";

export type ToastVariant = "neutral" | "success" | "warning" | "error" | "info";

export interface ToastOptions {
  title: ReactNode;
  description?: ReactNode;
  variant?: ToastVariant;
  /** Milliseconds before auto-dismiss; errors default to longer. 0 keeps it until closed. */
  duration?: number;
}

interface ToastItem extends Required<Pick<ToastOptions, "variant" | "duration">> {
  id: number;
  title: ReactNode;
  description?: ReactNode;
}

interface ToastContextValue {
  toast: (options: ToastOptions) => number;
  dismiss: (id: number) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

const MAX_VISIBLE = 4;

const icons: Record<ToastVariant, ReactNode> = {
  neutral: null,
  success: <CheckCircleIcon className="text-success" />,
  warning: <AlertTriangleIcon className="text-warning" />,
  error: <AlertCircleIcon className="text-error" />,
  info: <InfoIcon className="text-info" />,
};

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextId = useRef(1);

  const dismiss = useCallback((id: number) => {
    setToasts((current) => current.filter((item) => item.id !== id));
  }, []);

  const toast = useCallback((options: ToastOptions) => {
    const id = nextId.current++;
    const variant = options.variant ?? "neutral";
    const duration = options.duration ?? (variant === "error" ? 8000 : 5000);
    setToasts((current) => [...current, { ...options, id, variant, duration }].slice(-MAX_VISIBLE));
    return id;
  }, []);

  const value = useMemo(() => ({ toast, dismiss }), [toast, dismiss]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <section
        aria-label="Notifications"
        className="pointer-events-none fixed inset-x-4 bottom-4 z-(--z-toast) flex flex-col items-end gap-2 sm:left-auto sm:w-96"
      >
        {toasts.map((item) => (
          <Toast key={item.id} item={item} dismiss={dismiss} />
        ))}
      </section>
    </ToastContext.Provider>
  );
}

function Toast({ item, dismiss }: { item: ToastItem; dismiss: (id: number) => void }) {
  const [paused, setPaused] = useState(false);
  const onDismiss = useCallback(() => dismiss(item.id), [dismiss, item.id]);

  useEffect(() => {
    if (item.duration === 0 || paused) return;
    const timer = setTimeout(onDismiss, item.duration);
    return () => clearTimeout(timer);
  }, [item.duration, paused, onDismiss]);

  return (
    <div
      role={item.variant === "error" ? "alert" : "status"}
      aria-live={item.variant === "error" ? "assertive" : "polite"}
      onPointerEnter={() => setPaused(true)}
      onPointerLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)}
      onBlur={() => setPaused(false)}
      className="pointer-events-auto flex w-full animate-slide-in items-start gap-3 rounded-lg border border-border bg-surface p-4 shadow-lg"
    >
      {icons[item.variant] ? <span className="mt-0.5">{icons[item.variant]}</span> : null}
      <div className="flex flex-1 flex-col gap-0.5">
        <p className="text-sm font-medium text-fg">{item.title}</p>
        {item.description ? <p className="text-sm text-fg-muted">{item.description}</p> : null}
      </div>
      <IconButton
        aria-label="Dismiss notification"
        size="sm"
        onClick={onDismiss}
        className="-mt-1 -mr-1"
      >
        <XIcon />
      </IconButton>
    </div>
  );
}

export function useToast(): ToastContextValue {
  const context = useContext(ToastContext);
  if (!context) throw new Error("useToast must be used inside <ToastProvider>.");
  return context;
}
