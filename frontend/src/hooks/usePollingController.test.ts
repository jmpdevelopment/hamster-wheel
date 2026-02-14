import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useState } from "react";
import { usePollingController } from "./usePollingController";

const mockPollNow = vi.fn();
const mockGetPollingStatus = vi.fn();
const mockSetPollingPaused = vi.fn();
const mockEventsOn = vi.fn();

const noOpRun = {
  runID: "",
  startedAt: "",
  completedAt: "",
  durationMs: 0,
  totalFilters: 0,
  failedFilters: 0,
  newJobs: 0,
  skipped: 0,
  cycleError: "",
  diagnosticsPath: "",
  diagnosticsError: "",
  filters: [],
};

vi.mock("../../bindings/hamster-wheel/pollingservice", () => ({
  PollNow: (...args: unknown[]) => mockPollNow(...args),
  GetPollingStatus: (...args: unknown[]) => mockGetPollingStatus(...args),
  SetPollingPaused: (...args: unknown[]) => mockSetPollingPaused(...args),
}));

vi.mock("@wailsio/runtime", () => ({
  Create: {
    Array:
      (factory: (value: unknown) => unknown) =>
      (values: unknown) =>
        Array.isArray(values) ? values.map((value) => factory(value)) : [],
  },
  Events: {
    On: (...args: unknown[]) => mockEventsOn(...args),
  },
}));

function renderPollingController() {
  const refreshJobs = vi.fn().mockResolvedValue(undefined);
  const refreshFilters = vi.fn().mockResolvedValue(undefined);

  const hook = renderHook(() => {
    const [error, setError] = useState<string | null>(null);
    const polling = usePollingController({
      refreshJobs,
      refreshFilters,
      setAppError: setError,
    });
    return { polling, error };
  });

  return { ...hook, refreshJobs, refreshFilters };
}

beforeEach(() => {
  vi.clearAllMocks();
  mockPollNow.mockResolvedValue(noOpRun);
  mockGetPollingStatus.mockResolvedValue({ paused: false, nextPollAt: "" });
  mockSetPollingPaused.mockResolvedValue(undefined);
  mockEventsOn.mockImplementation(() => vi.fn());
});

describe("usePollingController", () => {
  it("surfaces scheduler startup poll run from status snapshot", async () => {
    mockGetPollingStatus.mockResolvedValue({
      paused: false,
      isPolling: false,
      nextPollAt: "2026-02-14T15:00:00Z",
      lastRun: {
        runID: "auto-123",
        startedAt: "2026-02-14T14:30:00Z",
        completedAt: "2026-02-14T14:30:10Z",
        durationMs: 0,
        totalFilters: 1,
        failedFilters: 0,
        newJobs: 3,
        skipped: 97,
        filters: [
          {
            filterID: "f1",
            filterName: "React developer",
            source: "reed_uk",
            newJobs: 3,
            skipped: 97,
          },
        ],
      },
    });

    const { result } = renderPollingController();

    await waitFor(() => {
      expect(result.current.polling.pollRun?.runID).toBe("auto-123");
      expect(result.current.polling.pollRun?.newJobs).toBe(3);
    });
  });

  it("blocks manual PollNow while scheduler polling is active", async () => {
    mockGetPollingStatus.mockResolvedValue({
      paused: false,
      isPolling: true,
      nextPollAt: "2026-02-14T15:00:00Z",
    });

    const { result } = renderPollingController();

    await waitFor(() => {
      expect(result.current.polling.isPolling).toBe(true);
    });

    await act(async () => {
      await result.current.polling.pollNow();
    });

    expect(mockPollNow).not.toHaveBeenCalled();
  });

  it("loads startup status and accepts PascalCase payloads", async () => {
    mockGetPollingStatus.mockResolvedValue({
      Paused: false,
      IsPolling: false,
      NextPollAt: "2026-02-14T14:30:00Z",
    });

    const { result } = renderPollingController();

    await waitFor(() => {
      expect(result.current.polling.pollingPaused).toBe(false);
      expect(result.current.polling.nextPollAt).toBe("2026-02-14T14:30:00Z");
    });
  });

  it("updates next poll from scheduler event payload without extra status request", async () => {
    const { result } = renderPollingController();

    await waitFor(() => {
      expect(mockGetPollingStatus).toHaveBeenCalledTimes(1);
    });

    const onStatusChanged = mockEventsOn.mock.calls.find(
      (call) => call[0] === "polling:status-changed"
    )?.[1] as ((event: unknown) => void) | undefined;

    expect(onStatusChanged).toBeTypeOf("function");

    act(() => {
      onStatusChanged?.({
        name: "polling:status-changed",
        data: { paused: false, nextPollAt: "2026-02-14T15:00:00Z" },
      });
    });

    expect(result.current.polling.nextPollAt).toBe("2026-02-14T15:00:00Z");
    expect(mockGetPollingStatus).toHaveBeenCalledTimes(1);
  });

  it("seeds a fallback next poll after manual poll when backend status is empty", async () => {
    const { result, refreshFilters, refreshJobs } = renderPollingController();

    await act(async () => {
      await result.current.polling.pollNow();
    });

    expect(mockPollNow).toHaveBeenCalledTimes(1);
    expect(refreshJobs).toHaveBeenCalled();
    expect(refreshFilters).toHaveBeenCalled();
    expect(result.current.polling.nextPollAt).not.toBe("");
  });

  it("updates next poll time after manual poll when backend returns a new schedule", async () => {
    const initialNext = "2026-02-14T14:00:00Z";
    const updatedNext = "2026-02-14T14:30:00Z";
    mockGetPollingStatus
      .mockResolvedValueOnce({ paused: false, nextPollAt: initialNext })
      .mockResolvedValue({ paused: false, nextPollAt: updatedNext });

    const { result } = renderPollingController();

    await waitFor(() => {
      expect(result.current.polling.nextPollAt).toBe(initialNext);
    });

    await act(async () => {
      await result.current.polling.pollNow();
    });

    expect(result.current.polling.nextPollAt).toBe(updatedNext);
  });

  it("seeds a fallback next poll when resuming and status is still empty", async () => {
    mockGetPollingStatus
      .mockResolvedValueOnce({ paused: true, nextPollAt: "" })
      .mockResolvedValue({ paused: false, nextPollAt: "" });

    const { result } = renderPollingController();

    await waitFor(() => {
      expect(result.current.polling.pollingPaused).toBe(true);
    });

    await act(async () => {
      await result.current.polling.togglePolling();
    });

    expect(mockSetPollingPaused).toHaveBeenCalledWith(false);
    expect(result.current.polling.pollingPaused).toBe(false);
    expect(result.current.polling.nextPollAt).not.toBe("");
  });

  it("refreshes jobs and status when window regains focus", async () => {
    const { refreshJobs } = renderPollingController();

    await waitFor(() => {
      expect(mockGetPollingStatus).toHaveBeenCalledTimes(1);
    });

    act(() => {
      window.dispatchEvent(new Event("focus"));
    });

    await waitFor(() => {
      expect(refreshJobs).toHaveBeenCalledTimes(1);
      expect(mockGetPollingStatus).toHaveBeenCalledTimes(2);
    });
  });
});
