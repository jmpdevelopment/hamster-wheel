import { Button } from "./Button";
import { IconButton } from "./IconButton";

interface HeaderProps {
  jobCount: number;
  onPollNow: () => void;
  isPolling: boolean;
  pollingPaused: boolean;
  nextPollAt: string; // RFC3339 timestamp or empty
  onTogglePolling: () => void;
  hasFilters: boolean;
  hasEnabledFilters: boolean;
  onOpenSettings: () => void;
}

function formatNextPoll(isoString: string): string {
  if (!isoString) return "";
  const date = new Date(isoString);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function Header({
  jobCount,
  onPollNow,
  isPolling,
  pollingPaused,
  nextPollAt,
  onTogglePolling,
  hasFilters,
  hasEnabledFilters,
  onOpenSettings,
}: HeaderProps) {
  const canPoll = hasFilters && hasEnabledFilters;

  const statusText = !hasFilters
    ? "Add a filter to start monitoring"
    : !hasEnabledFilters
      ? "Enable a filter to start monitoring"
      : pollingPaused
        ? "Auto-poll paused"
        : nextPollAt
          ? `Next: ${formatNextPoll(nextPollAt)}`
          : "";

  const disabledReason = !hasFilters
    ? "No filters configured"
    : !hasEnabledFilters
      ? "No filters enabled"
      : undefined;

  return (
    <header className="flex items-center justify-between px-4 py-3 border-b border-hw-border bg-hw-bg">
      <h1 className="text-xl font-bold text-hw-accent">Hamster Wheel</h1>

      <div className="flex items-center gap-4">
        <span className="text-sm text-hw-text-muted">
          {jobCount} {jobCount === 1 ? "job" : "jobs"}
        </span>

        <span className="text-xs text-hw-text-muted">
          {statusText}
        </span>

        <IconButton aria-label="Open settings" onClick={onOpenSettings}>
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
          >
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z" />
          </svg>
        </IconButton>

        <Button
          variant="secondary"
          size="sm"
          onClick={onTogglePolling}
          disabled={!canPoll}
          aria-label={
            !canPoll
              ? disabledReason
              : pollingPaused
                ? "Resume auto-polling"
                : "Pause auto-polling"
          }
          title={!canPoll ? disabledReason : undefined}
        >
          {pollingPaused ? "Resume" : "Pause"}
        </Button>

        <Button
          variant="primary"
          size="md"
          onClick={onPollNow}
          disabled={!canPoll}
          loading={isPolling}
          title={!canPoll ? disabledReason : undefined}
        >
          {isPolling ? "Polling..." : "Poll Now"}
        </Button>
      </div>
    </header>
  );
}
