import { PollRunResult } from "../../bindings/hamster-wheel/models";
import { SavePollReport } from "../../bindings/hamster-wheel/pollingservice";
import { Dialogs } from "@wailsio/runtime";

function pluralize(value: number, singular: string, plural: string): string {
  return value === 1 ? singular : plural;
}

// Build a plain-text report users can attach when reporting polling failures.
export function buildPollReport(run: PollRunResult): string {
  const lines: string[] = [
    "Hamster Wheel Poll Report",
    `Run ID: ${run.runID || "n/a"}`,
    `Started: ${run.startedAt || "n/a"}`,
    `Completed: ${run.completedAt || "n/a"}`,
    `Duration: ${run.durationMs ?? 0} ms`,
    `Totals: ${run.newJobs ?? 0} new, ${run.skipped ?? 0} skipped, ${run.failedFilters ?? 0} failed filters`,
    `Filters polled: ${run.totalFilters ?? 0}`,
  ];

  if (run.cycleError) {
    lines.push(`Cycle error: ${run.cycleError}`);
  }
  if (run.diagnosticsPath) {
    lines.push(`Diagnostics file: ${run.diagnosticsPath}`);
  }
  if (run.diagnosticsError) {
    lines.push(`Diagnostics write error: ${run.diagnosticsError}`);
  }

  lines.push("");
  lines.push("Filter outcomes:");

  const filters = run.filters ?? [];
  if (filters.length === 0) {
    lines.push("  (no filter-level outcomes)");
  } else {
    for (const filter of filters) {
      const name = filter.filterName || filter.filterID || "Unknown filter";
      if (filter.error) {
        lines.push(`  - ${name}: ERROR: ${filter.error}`);
      } else {
        lines.push(
          `  - ${name}: ${filter.newJobs ?? 0} new, ${filter.skipped ?? 0} skipped`
        );
      }
    }
  }

  lines.push("");
  lines.push(
    `Summary: ${run.failedFilters ?? 0} ${pluralize(run.failedFilters ?? 0, "filter failed", "filters failed")}`
  );
  return `${lines.join("\n")}\n`;
}

export async function downloadPollReport(run: PollRunResult): Promise<boolean> {
  const report = buildPollReport(run);
  const filename = `hamster-wheel-poll-report-${run.runID || "unknown"}.txt`;

  const path = await Dialogs.SaveFile({
    Title: "Save Poll Report",
    Filename: filename,
    Filters: [
      {
        DisplayName: "Text files",
        Pattern: "*.txt",
      },
    ],
  });
  if (!path) {
    return false;
  }

  await SavePollReport(path, report);
  return true;
}
