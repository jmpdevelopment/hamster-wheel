interface ErrorBannerProps {
  message: string | null;
  onDismiss: () => void;
}

export function ErrorBanner({ message, onDismiss }: ErrorBannerProps) {
  if (!message) return null;

  return (
    <div className="flex items-center justify-between gap-3 bg-hw-danger/20 border border-hw-danger/40 text-hw-danger px-4 py-2 text-sm">
      <span>{message}</span>
      <button
        onClick={onDismiss}
        className="shrink-0 text-hw-danger hover:text-white font-bold"
        aria-label="Dismiss error"
      >
        ✕
      </button>
    </div>
  );
}
