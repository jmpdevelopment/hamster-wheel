import { useEffect } from "react";
import { PollResult } from "../../bindings/hamster-wheel/internal/scheduler/models";

interface PollResultToastProps {
  results: PollResult[] | null;
  onDismiss: () => void;
}

export function PollResultToast({ results, onDismiss }: PollResultToastProps) {
  useEffect(() => {
    if (!results) return;
    const timer = setTimeout(onDismiss, 5000);
    return () => clearTimeout(timer);
  }, [results, onDismiss]);

  if (!results) return null;

  const totalNew = results.reduce((sum, r) => sum + (r.NewJobs ?? 0), 0);
  const totalSkipped = results.reduce((sum, r) => sum + (r.Skipped ?? 0), 0);

  return (
    <div className="fixed bottom-4 right-4 bg-hw-surface border border-hw-border rounded-lg shadow-lg p-4 max-w-sm z-50">
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className="text-sm font-semibold text-hw-text">
            Poll complete: {totalNew} new, {totalSkipped} skipped
          </p>
          <ul className="mt-2 space-y-1">
            {results.map((r) => (
              <li key={r.FilterID} className="text-xs text-hw-text-muted">
                {r.FilterName}:{" "}
                {r.Err ? (
                  <span className="text-hw-danger">error</span>
                ) : (
                  <span>
                    {r.NewJobs ?? 0} new, {r.Skipped ?? 0} skipped
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
        <button
          onClick={onDismiss}
          className="shrink-0 text-hw-text-muted hover:text-hw-text text-sm"
          aria-label="Dismiss"
        >
          ✕
        </button>
      </div>
    </div>
  );
}
