import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, afterEach } from "vitest";
import { PollResultToast } from "./PollResultToast";

const mockDownloadPollReport = vi.fn();
vi.mock("../lib/pollReport", () => ({
  downloadPollReport: (...args: unknown[]) => mockDownloadPollReport(...args),
}));

const fakeRun = {
  runID: "run-1",
  startedAt: "2026-02-14T10:00:00Z",
  completedAt: "2026-02-14T10:00:02Z",
  durationMs: 2000,
  totalFilters: 2,
  failedFilters: 0,
  newJobs: 4,
  skipped: 2,
  cycleError: "",
  diagnosticsPath: "/tmp/report.json",
  diagnosticsError: "",
  filters: [
    { filterID: "f1", filterName: "Backend London", source: "reed_uk", newJobs: 3, skipped: 2, error: "" },
    { filterID: "f2", filterName: "Remote React", source: "reed_uk", newJobs: 1, skipped: 0, error: "" },
  ],
};

afterEach(() => {
  vi.clearAllMocks();
  vi.useRealTimers();
});

describe("PollResultToast", () => {
  it("renders nothing when run is null", () => {
    const { container } = render(
      <PollResultToast run={null} onDismiss={() => {}} />
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders nothing when run is a no-op poll", () => {
    const { container } = render(
      <PollResultToast
        run={{
          ...fakeRun,
          totalFilters: 0,
          failedFilters: 0,
          cycleError: "",
          newJobs: 0,
          skipped: 0,
          filters: [],
        }}
        onDismiss={() => {}}
      />
    );
    expect(container.firstChild).toBeNull();
  });

  it("shows total new and skipped", () => {
    render(<PollResultToast run={fakeRun} onDismiss={() => {}} />);
    expect(screen.getByText("Poll complete: 4 new, 2 skipped")).toBeInTheDocument();
  });

  it("shows per-filter results", () => {
    render(<PollResultToast run={fakeRun} onDismiss={() => {}} />);
    expect(screen.getByText(/Backend London/)).toBeInTheDocument();
    expect(screen.getByText(/Remote React/)).toBeInTheDocument();
  });

  it("shows error for failed filter", () => {
    const withError = {
      ...fakeRun,
      failedFilters: 1,
      newJobs: 0,
      skipped: 0,
      filters: [
        { filterID: "f1", filterName: "Broken", source: "reed_uk", newJobs: 0, skipped: 0, error: "timeout" },
      ],
    };
    render(<PollResultToast run={withError} onDismiss={() => {}} />);
    expect(
      screen.getByText("Poll complete: 0 new, 0 skipped, 1 failed")
    ).toBeInTheDocument();
    expect(screen.getByText("error: timeout")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save report/i })).toBeInTheDocument();
  });

  it("shows cycle error for complete failures", () => {
    const failedRun = {
      ...fakeRun,
      totalFilters: 0,
      failedFilters: 0,
      newJobs: 0,
      skipped: 0,
      cycleError: "listing enabled filters: database unavailable",
      filters: [],
    };
    render(<PollResultToast run={failedRun} onDismiss={() => {}} />);
    expect(screen.getByText("Poll failed")).toBeInTheDocument();
    expect(screen.getByText(/cycle error:/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save report/i })).toBeInTheDocument();
  });

  it("triggers report export when Save report is clicked", async () => {
    const withError = {
      ...fakeRun,
      failedFilters: 1,
      filters: [
        { filterID: "f1", filterName: "Broken", source: "reed_uk", newJobs: 0, skipped: 0, error: "timeout" },
      ],
    };
    render(<PollResultToast run={withError} onDismiss={() => {}} />);

    await userEvent.click(screen.getByRole("button", { name: /save report/i }));
    expect(mockDownloadPollReport).toHaveBeenCalledOnce();
  });

  it("calls onDismiss when dismiss button is clicked", async () => {
    const onDismiss = vi.fn();
    render(<PollResultToast run={fakeRun} onDismiss={onDismiss} />);

    await userEvent.click(screen.getByRole("button", { name: /dismiss/i }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("auto-dismisses after 5 seconds", () => {
    vi.useFakeTimers();
    const onDismiss = vi.fn();
    render(<PollResultToast run={fakeRun} onDismiss={onDismiss} />);

    expect(onDismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(5000);
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("does not auto-dismiss when failures exist", () => {
    vi.useFakeTimers();
    const onDismiss = vi.fn();
    const withError = {
      ...fakeRun,
      failedFilters: 1,
      filters: [
        { filterID: "f1", filterName: "Broken", source: "reed_uk", newJobs: 0, skipped: 0, error: "timeout" },
      ],
    };
    render(<PollResultToast run={withError} onDismiss={onDismiss} />);

    vi.advanceTimersByTime(10000);
    expect(onDismiss).not.toHaveBeenCalled();
  });
});
