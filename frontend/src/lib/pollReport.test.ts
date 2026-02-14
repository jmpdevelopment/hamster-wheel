import { describe, it, expect, vi, beforeEach } from "vitest";

const mockSaveFile = vi.fn();
const mockSavePollReport = vi.fn();

vi.mock("@wailsio/runtime", () => ({
  Dialogs: {
    SaveFile: (...args: unknown[]) => mockSaveFile(...args),
  },
}));

vi.mock("../../bindings/hamster-wheel/pollingservice", () => ({
  SavePollReport: (...args: unknown[]) => mockSavePollReport(...args),
}));

import { buildPollReport, downloadPollReport } from "./pollReport";

const sampleRun = {
  runID: "run-123",
  startedAt: "2026-02-14T10:00:00Z",
  completedAt: "2026-02-14T10:00:05Z",
  durationMs: 5000,
  totalFilters: 2,
  failedFilters: 1,
  newJobs: 3,
  skipped: 4,
  cycleError: "",
  diagnosticsPath: "/tmp/poll-run-run-123.json",
  diagnosticsError: "",
  filters: [
    {
      filterID: "f1",
      filterName: "Backend",
      source: "reed_uk",
      newJobs: 3,
      skipped: 1,
      error: "",
    },
    {
      filterID: "f2",
      filterName: "Remote",
      source: "reed_uk",
      newJobs: 0,
      skipped: 3,
      error: "upstream timeout",
    },
  ],
};

beforeEach(() => {
  vi.clearAllMocks();
  mockSaveFile.mockResolvedValue("");
  mockSavePollReport.mockResolvedValue(undefined);
});

describe("buildPollReport", () => {
  it("formats a report with filter outcomes and cycle metadata", () => {
    const report = buildPollReport(sampleRun);

    expect(report).toContain("Hamster Wheel Poll Report");
    expect(report).toContain("Run ID: run-123");
    expect(report).toContain("Totals: 3 new, 4 skipped, 1 failed filters");
    expect(report).toContain("Diagnostics file: /tmp/poll-run-run-123.json");
    expect(report).toContain("Backend: 3 new, 1 skipped");
    expect(report).toContain("Remote: ERROR: upstream timeout");
    expect(report).toContain("Summary: 1 filter failed");
  });

  it("formats complete failures with no filter outcomes", () => {
    const report = buildPollReport({
      runID: "run-fail",
      startedAt: "2026-02-14T10:00:00Z",
      completedAt: "2026-02-14T10:00:01Z",
      durationMs: 1000,
      totalFilters: 0,
      failedFilters: 0,
      newJobs: 0,
      skipped: 0,
      cycleError: "listing enabled filters: database unavailable",
      diagnosticsPath: "",
      diagnosticsError: "cleaning diagnostics: permission denied",
      filters: [],
    });

    expect(report).toContain("Cycle error: listing enabled filters: database unavailable");
    expect(report).toContain("Diagnostics write error: cleaning diagnostics: permission denied");
    expect(report).toContain("(no filter-level outcomes)");
    expect(report).toContain("Summary: 0 filters failed");
  });

  it("opens save dialog and writes report when user confirms", async () => {
    mockSaveFile.mockResolvedValue("/tmp/poll-report.txt");

    const saved = await downloadPollReport(sampleRun);

    expect(saved).toBe(true);
    expect(mockSaveFile).toHaveBeenCalledOnce();
    expect(mockSavePollReport).toHaveBeenCalledOnce();
    expect(mockSavePollReport.mock.calls[0][0]).toBe("/tmp/poll-report.txt");
    expect(String(mockSavePollReport.mock.calls[0][1])).toContain("Run ID: run-123");
  });

  it("returns false when user cancels save dialog", async () => {
    mockSaveFile.mockResolvedValue("");

    const saved = await downloadPollReport(sampleRun);

    expect(saved).toBe(false);
    expect(mockSavePollReport).not.toHaveBeenCalled();
  });
});
