import { useEffect } from "react";
import { IconButton } from "./IconButton";

interface ToastProps {
  variant?: "info" | "success" | "error";
  title: string;
  children?: React.ReactNode;
  duration?: number;
  onDismiss: () => void;
}

const variantConfig = {
  info: {
    containerClass: "border-hw-border",
    iconColor: "text-hw-accent",
    icon: (
      <svg
        width="18"
        height="18"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="16" x2="12" y2="12" />
        <line x1="12" y1="8" x2="12.01" y2="8" />
      </svg>
    ),
  },
  success: {
    containerClass: "border-hw-success/50",
    iconColor: "text-hw-success",
    icon: (
      <svg
        width="18"
        height="18"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="M8 12l3 3 5-5" />
      </svg>
    ),
  },
  error: {
    containerClass: "border-hw-danger/50",
    iconColor: "text-hw-danger",
    icon: (
      <svg
        width="18"
        height="18"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
        <line x1="12" y1="9" x2="12" y2="13" />
        <line x1="12" y1="17" x2="12.01" y2="17" />
      </svg>
    ),
  },
} as const;

export function Toast({
  variant = "info",
  title,
  children,
  duration = 5000,
  onDismiss,
}: ToastProps) {
  useEffect(() => {
    if (duration <= 0) return;
    const timer = setTimeout(onDismiss, duration);
    return () => clearTimeout(timer);
  }, [duration, onDismiss]);

  const config = variantConfig[variant];

  return (
    <div
      role={variant === "error" ? "alert" : "status"}
      className={`fixed bottom-4 right-4 z-50 max-w-sm rounded-lg border bg-hw-surface shadow-lg p-4 transition-opacity duration-200 ${config.containerClass}`}
    >
      <div className="flex items-start gap-3">
        <span className={`shrink-0 mt-0.5 ${config.iconColor}`}>
          {config.icon}
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-hw-text">{title}</p>
          {children && (
            <div className="mt-1 text-xs text-hw-text-muted leading-relaxed">
              {children}
            </div>
          )}
        </div>
        <IconButton
          aria-label="Dismiss"
          onClick={onDismiss}
          className="shrink-0 -mt-1 -mr-1"
        >
          ✕
        </IconButton>
      </div>
    </div>
  );
}
