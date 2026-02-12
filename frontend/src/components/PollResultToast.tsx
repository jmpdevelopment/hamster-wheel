import { PollResult } from "../../bindings/hamster-wheel/internal/scheduler/models";
import { Toast } from "./Toast";

interface PollResultToastProps {
  results: PollResult[] | null;
  onDismiss: () => void;
}

export function PollResultToast({ results, onDismiss }: PollResultToastProps) {
  if (!results) return null;

  const totalNew = results.reduce((sum, r) => sum + (r.NewJobs ?? 0), 0);
  const totalSkipped = results.reduce((sum, r) => sum + (r.Skipped ?? 0), 0);

  return (
    <Toast
      variant={totalNew > 0 ? "success" : "info"}
      title={`Poll complete: ${totalNew} new, ${totalSkipped} skipped`}
      duration={5000}
      onDismiss={onDismiss}
    >
      <ul className="space-y-1">
        {results.map((r) => (
          <li key={r.FilterID}>
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
    </Toast>
  );
}
