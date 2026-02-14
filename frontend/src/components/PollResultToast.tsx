import { useState } from "react";
import { PollRunResult } from "../../bindings/hamster-wheel/models";
import { downloadPollReport } from "../lib/pollReport";
import { Button } from "./Button";
import { Toast } from "./Toast";

interface PollResultToastProps {
  run: PollRunResult | null;
  onDismiss: () => void;
}

export function PollResultToast({ run, onDismiss }: PollResultToastProps) {
  const [savingReport, setSavingReport] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  if (!run) return null;
  if ((run.totalFilters ?? 0) === 0 && !run.cycleError) return null;

  const totalNew = run.newJobs ?? 0;
  const totalSkipped = run.skipped ?? 0;
  const totalFailed = run.failedFilters ?? 0;
  const hasCycleError = Boolean(run.cycleError);
  const hasFailures = totalFailed > 0 || hasCycleError;
  const summaryTitle = hasCycleError
    ? "Poll failed"
    : hasFailures
      ? `Poll complete: ${totalNew} new, ${totalSkipped} skipped, ${totalFailed} failed`
      : `Poll complete: ${totalNew} new, ${totalSkipped} skipped`;

  return (
    <Toast
      variant={hasFailures ? "error" : totalNew > 0 ? "success" : "info"}
      title={summaryTitle}
      duration={hasFailures ? 0 : 5000}
      onDismiss={onDismiss}
    >
      {run.cycleError && (
        <p className="mb-1">
          <span className="text-hw-danger">cycle error: {run.cycleError}</span>
        </p>
      )}
      {run.diagnosticsError && (
        <p className="mb-1">
          diagnostics error: {run.diagnosticsError}
        </p>
      )}
      <ul className="space-y-1">
        {(run.filters ?? []).map((filter) => (
          <li key={filter.filterID}>
            {filter.filterName}:{" "}
            {filter.error ? (
              <span className="text-hw-danger">error: {filter.error}</span>
            ) : (
              <span>
                {filter.newJobs ?? 0} new, {filter.skipped ?? 0} skipped
              </span>
            )}
          </li>
        ))}
      </ul>
      {hasFailures && (
        <div className="mt-3">
          <Button
            type="button"
            size="sm"
            variant="secondary"
            loading={savingReport}
            onClick={async () => {
              setSaveError(null);
              setSavingReport(true);
              try {
                await downloadPollReport(run);
              } catch (err: unknown) {
                const message = err instanceof Error ? err.message : String(err);
                setSaveError(message);
              } finally {
                setSavingReport(false);
              }
            }}
          >
            Save report
          </Button>
          {saveError && (
            <p className="mt-1 text-hw-danger">
              report save failed: {saveError}
            </p>
          )}
        </div>
      )}
    </Toast>
  );
}
