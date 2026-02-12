import { Button } from "./Button";

interface HeaderProps {
  jobCount: number;
  onPollNow: () => void;
  isPolling: boolean;
  pollingPaused: boolean;
  nextPollAt: string; // RFC3339 timestamp or empty
  onTogglePolling: () => void;
  hasFilters: boolean;
  hasEnabledFilters: boolean;
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
