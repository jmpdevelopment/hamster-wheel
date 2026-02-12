import { IconButton } from "./IconButton";

interface ErrorBannerProps {
  message: string | null;
  onDismiss: () => void;
}

export function ErrorBanner({ message, onDismiss }: ErrorBannerProps) {
  if (!message) return null;

  return (
    <div
      role="alert"
      className="flex items-center justify-between gap-3 bg-hw-danger/20 border border-hw-danger/40 text-hw-danger px-4 py-2 text-sm"
    >
      <span className="flex items-center gap-2 leading-relaxed">
        <svg
          aria-hidden="true"
          className="shrink-0 w-4 h-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
          <line x1="12" y1="9" x2="12" y2="13" />
          <line x1="12" y1="17" x2="12.01" y2="17" />
        </svg>
        {message}
      </span>
      <IconButton
        aria-label="Dismiss error"
        onClick={onDismiss}
        className="shrink-0 text-hw-danger"
      >
        ✕
      </IconButton>
    </div>
  );
}
