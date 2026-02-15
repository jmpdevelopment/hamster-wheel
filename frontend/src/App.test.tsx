import { render, screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import App from "./App";

const fakeJob = (id = "j1", title = "Go Developer") => ({
  ID: id,
  Source: "reed_uk",
  SourceID: `src-${id}`,
  Title: title,
  Company: "Acme",
  Location: "London",
  Description: "A job",
  URL: "https://example.com",
  PostedAt: "2026-02-08T10:00:00Z",
  DiscoveredAt: "2026-02-08T11:00:00Z",
  FilterID: "f1",
  IsFavorite: false,
});

const mockGetJobs = vi.fn();
const mockGetJobCount = vi.fn();
const mockDeleteJob = vi.fn();
const mockSetJobFavorite = vi.fn();
const mockSetJobsFavorite = vi.fn();
const mockRecalculateMatchScore = vi.fn();
const mockGetFilters = vi.fn();
const mockCreateFilter = vi.fn();
const mockUpdateFilter = vi.fn();
const mockDeleteFilter = vi.fn();
const mockPollNow = vi.fn();
const mockGetPollingStatus = vi.fn();
const mockSetPollingPaused = vi.fn();
const mockSetPollingIntervalMinutes = vi.fn();
const mockGetKeyboardShortcuts = vi.fn();
const mockGetFirstRunComplete = vi.fn();
const mockSetFirstRunComplete = vi.fn();
const mockHasReedAPIKey = vi.fn();
const mockHasAdzunaCredentials = vi.fn();
const mockGetJobListPreferences = vi.fn();
const mockSetJobListPreferences = vi.fn();
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

// Mock all Wails bindings.
vi.mock("../bindings/hamster-wheel/jobservice", () => ({
  GetJobs: (...args: unknown[]) => mockGetJobs(...args),
  GetJobCount: (...args: unknown[]) => mockGetJobCount(...args),
  DeleteJob: (...args: unknown[]) => mockDeleteJob(...args),
  SetJobFavorite: (...args: unknown[]) => mockSetJobFavorite(...args),
  SetJobsFavorite: (...args: unknown[]) => mockSetJobsFavorite(...args),
  RecalculateMatchScore: (...args: unknown[]) =>
    mockRecalculateMatchScore(...args),
}));

vi.mock("../bindings/hamster-wheel/filterservice", () => ({
  GetFilters: (...args: unknown[]) => mockGetFilters(...args),
  CreateFilter: (...args: unknown[]) => mockCreateFilter(...args),
  UpdateFilter: (...args: unknown[]) => mockUpdateFilter(...args),
  DeleteFilter: (...args: unknown[]) => mockDeleteFilter(...args),
}));

vi.mock("../bindings/hamster-wheel/pollingservice", () => ({
  PollNow: (...args: unknown[]) => mockPollNow(...args),
  GetPollingStatus: (...args: unknown[]) => mockGetPollingStatus(...args),
  SetPollingPaused: (...args: unknown[]) => mockSetPollingPaused(...args),
  SetPollingIntervalMinutes: (...args: unknown[]) =>
    mockSetPollingIntervalMinutes(...args),
}));

vi.mock("../bindings/hamster-wheel/settingsservice", () => ({
  GetFirstRunComplete: (...args: unknown[]) => mockGetFirstRunComplete(...args),
  SetFirstRunComplete: (...args: unknown[]) => mockSetFirstRunComplete(...args),
  HasReedAPIKey: (...args: unknown[]) => mockHasReedAPIKey(...args),
  SetReedAPIKey: vi.fn().mockResolvedValue(undefined),
  ClearReedAPIKey: vi.fn().mockResolvedValue(undefined),
  HasAdzunaCredentials: (...args: unknown[]) =>
    mockHasAdzunaCredentials(...args),
  SetAdzunaCredentials: vi.fn().mockResolvedValue(undefined),
  ClearAdzunaCredentials: vi.fn().mockResolvedValue(undefined),
  HasOpenAIAPIKey: vi.fn().mockResolvedValue(false),
  SetOpenAIAPIKey: vi.fn().mockResolvedValue(undefined),
  ClearOpenAIAPIKey: vi.fn().mockResolvedValue(undefined),
  GetLLMMode: vi.fn().mockResolvedValue("cloud"),
  SetLLMMode: vi.fn().mockResolvedValue(undefined),
  SetLLMProvider: vi.fn().mockResolvedValue(undefined),
  GetLLMModel: vi.fn().mockResolvedValue("gpt-4o-mini"),
  SetLLMModel: vi.fn().mockResolvedValue(undefined),
  GetAutoMatchEnabled: vi.fn().mockResolvedValue(true),
  SetAutoMatchEnabled: vi.fn().mockResolvedValue(undefined),
  GetAutoMatchLimit: vi.fn().mockResolvedValue(0),
  SetAutoMatchLimit: vi.fn().mockResolvedValue(undefined),
  GetAutoPollingEnabled: vi.fn().mockResolvedValue(false),
  SetAutoPollingEnabled: vi.fn().mockResolvedValue(undefined),
  GetPollIntervalMinutes: vi.fn().mockResolvedValue(30),
  SetPollIntervalMinutes: vi.fn().mockResolvedValue(undefined),
  GetJobRetentionDays: vi.fn().mockResolvedValue(30),
  SetJobRetentionDays: vi.fn().mockResolvedValue(undefined),
  GetLocalRuntimeModel: vi.fn().mockResolvedValue("llama3.1:8b"),
  GetLocalRuntimeStatus: vi.fn().mockResolvedValue({
    status: "ready",
    message: "",
    startedByApp: false,
  }),
  GetLocalRuntimeModels: vi.fn().mockResolvedValue({
    installed: [{ name: "llama3.1:8b" }],
  }),
  GetLocalRuntimePullProgress: vi.fn().mockResolvedValue({
    active: false,
    model: "llama3.1:8b",
    status: "",
    message: "",
    totalBytes: 0,
    completedBytes: 0,
    percent: 0,
    ready: false,
  }),
  PullLocalRuntimeModel: vi.fn().mockResolvedValue({
    model: "llama3.1:8b",
    ready: true,
    status: "success",
  }),
  StartLocalRuntime: vi.fn().mockResolvedValue({
    status: "ready",
    message: "",
    startedByApp: true,
  }),
  StopLocalRuntime: vi.fn().mockResolvedValue({
    status: "stopped",
    message: "",
    startedByApp: false,
  }),
  SetLocalRuntimeEngine: vi.fn().mockResolvedValue(undefined),
  SetLocalRuntimeModel: vi.fn().mockResolvedValue(undefined),
  GetCVPath: vi.fn().mockResolvedValue(""),
  SetCVPath: vi.fn().mockResolvedValue(undefined),
  GetTheme: vi.fn().mockResolvedValue(""),
  SetTheme: vi.fn().mockResolvedValue(undefined),
  GetKeyboardShortcuts: (...args: unknown[]) =>
    mockGetKeyboardShortcuts(...args),
  SetKeyboardShortcuts: vi.fn().mockResolvedValue(undefined),
  GetJobListPreferences: (...args: unknown[]) =>
    mockGetJobListPreferences(...args),
  SetJobListPreferences: (...args: unknown[]) =>
    mockSetJobListPreferences(...args),
}));

vi.mock("@wailsio/runtime", () => ({
  Browser: { OpenURL: vi.fn() },
  Dialogs: {
    SaveFile: vi.fn().mockResolvedValue(""),
    OpenFile: vi.fn().mockResolvedValue(""),
  },
  Create: {
    Array:
      (factory: (value: unknown) => unknown) =>
      (values: unknown) =>
        Array.isArray(values) ? values.map((value) => factory(value)) : [],
    Nullable:
      (factory: (value: unknown) => unknown) =>
      (value: unknown) =>
        value == null ? null : factory(value),
  },
  Events: {
    On: (...args: unknown[]) => mockEventsOn(...args),
  },
}));

vi.mock("react-virtualized-auto-sizer", () => ({
  AutoSizer: ({
    renderProp,
  }: {
    renderProp: (size: { height: number; width: number }) => React.ReactNode;
  }) => renderProp({ height: 600, width: 400 }),
}));

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  mockGetJobs.mockResolvedValue([]);
  mockGetJobCount.mockResolvedValue(0);
  mockDeleteJob.mockResolvedValue(undefined);
  mockSetJobFavorite.mockResolvedValue(undefined);
  mockSetJobsFavorite.mockResolvedValue(undefined);
  mockRecalculateMatchScore.mockResolvedValue(undefined);
  mockGetFilters.mockResolvedValue([]);
  mockCreateFilter.mockResolvedValue("f1");
  mockUpdateFilter.mockResolvedValue(undefined);
  mockDeleteFilter.mockResolvedValue(undefined);
  mockPollNow.mockResolvedValue(noOpRun);
  mockGetPollingStatus.mockResolvedValue({ paused: false, nextPollAt: "" });
  mockSetPollingPaused.mockResolvedValue(undefined);
  mockSetPollingIntervalMinutes.mockResolvedValue(undefined);
  mockGetKeyboardShortcuts.mockResolvedValue("");
  mockGetFirstRunComplete.mockResolvedValue(true);
  mockSetFirstRunComplete.mockResolvedValue(undefined);
  mockHasReedAPIKey.mockResolvedValue(false);
  mockHasAdzunaCredentials.mockResolvedValue(false);
  mockGetJobListPreferences.mockResolvedValue({
    filterByFilterId: "",
    sortMode: "posted-desc",
    postedDateFilterMode: "any",
    matchScoreFilterMode: "any",
    showFavoritesOnly: false,
  });
  mockSetJobListPreferences.mockResolvedValue(undefined);
  mockEventsOn.mockImplementation(() => vi.fn());
});

describe("App", () => {
  it("renders the app title and layout", async () => {
    render(<App />);

    expect(screen.getByText("Hamster Wheel")).toBeInTheDocument();
    expect(screen.getByText("Poll Now")).toBeInTheDocument();

    // Wait for hooks to finish loading.
    await waitFor(() => {
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });
  });

  it("shows empty states when no data", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("No filters")).toBeInTheDocument();
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });
  });

  it("shows initial setup wizard for first launch when no job provider is configured", async () => {
    mockGetFirstRunComplete.mockResolvedValue(false);
    mockHasReedAPIKey.mockResolvedValue(false);
    mockHasAdzunaCredentials.mockResolvedValue(false);

    render(<App />);

    await waitFor(() => {
      expect(
        screen.getByRole("dialog", { name: "Initial setup wizard" })
      ).toBeInTheDocument();
    });
  });

  it("does not auto-complete onboarding from provider credentials", async () => {
    mockGetFirstRunComplete.mockResolvedValue(false);
    mockHasReedAPIKey.mockResolvedValue(true);
    mockHasAdzunaCredentials.mockResolvedValue(false);

    render(<App />);

    await waitFor(() => {
      expect(
        screen.getByRole("dialog", { name: "Initial setup wizard" })
      ).toBeInTheDocument();
    });
    expect(mockSetFirstRunComplete).not.toHaveBeenCalled();
  });

  it("shows startup error when loading polling status fails", async () => {
    mockGetPollingStatus.mockRejectedValue(new Error("status load failed"));

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("status load failed")).toBeInTheDocument();
    });
  });

  it("shows startup error when loading keyboard shortcuts fails", async () => {
    mockGetKeyboardShortcuts.mockRejectedValue(
      new Error("shortcuts load failed")
    );

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("shortcuts load failed")).toBeInTheDocument();
    });
  });

  it("shows poll toast on startup when first scheduler poll already completed", async () => {
    mockGetFilters.mockResolvedValue([
      {
        ID: "f1",
        Name: "Backend",
        Keywords: "go",
        Location: "London",
        Source: "reed_uk",
        Enabled: true,
        CreatedAt: "2026-02-08T10:00:00Z",
        UpdatedAt: "2026-02-08T10:00:00Z",
      },
    ]);
    mockGetPollingStatus.mockResolvedValue({
      paused: false,
      isPolling: false,
      nextPollAt: "2026-02-14T16:08:40Z",
      lastRun: {
        runID: "auto-123",
        startedAt: "2026-02-14T15:38:40Z",
        completedAt: "2026-02-14T15:38:46Z",
        durationMs: 0,
        totalFilters: 1,
        failedFilters: 0,
        newJobs: 3,
        skipped: 97,
        filters: [
          {
            filterID: "f1",
            filterName: "Backend",
            source: "reed_uk",
            newJobs: 3,
            skipped: 97,
          },
        ],
      },
    });

    render(<App />);

    await waitFor(() => {
      expect(
        screen.getByText("Poll complete: 3 new, 97 skipped")
      ).toBeInTheDocument();
    });
  });

  it("uses scheduler event payload to update next poll without extra status call", async () => {
    mockGetFilters.mockResolvedValue([
      {
        ID: "f1",
        Name: "Backend",
        Keywords: "go",
        Location: "London",
        Source: "reed_uk",
        Enabled: true,
        CreatedAt: "2026-02-08T10:00:00Z",
        UpdatedAt: "2026-02-08T10:00:00Z",
      },
    ]);
    mockGetPollingStatus.mockResolvedValue({ paused: false, nextPollAt: "" });

    render(<App />);

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
        data: { paused: false, nextPollAt: "2026-02-14T11:40:30Z" },
      });
    });

    await waitFor(() => {
      expect(screen.getByText(/Next:/)).toBeInTheDocument();
    });
    expect(mockGetPollingStatus).toHaveBeenCalledTimes(1);
  });

  it("dismisses hook errors from the banner", async () => {
    mockGetJobs.mockRejectedValue(new Error("network down"));

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("network down")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByRole("button", { name: /dismiss/i }));

    await waitFor(() => {
      expect(screen.queryByText("network down")).not.toBeInTheDocument();
    });
  });

  it("keeps job detail open when delete fails", async () => {
    mockGetJobs.mockResolvedValue([fakeJob()]);
    mockGetJobCount.mockResolvedValue(1);
    mockDeleteJob.mockRejectedValue(new Error("cannot delete"));

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("Go Developer")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText("Go Developer"));
    await userEvent.click(screen.getByRole("button", { name: /delete job/i }));
    await userEvent.click(
      screen.getByRole("button", { name: /confirm delete/i })
    );

    await waitFor(() => {
      expect(screen.getByText("cannot delete")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: /close detail/i })
    ).toBeInTheDocument();
  });

  it("shows complete poll failure in toast with report action", async () => {
    mockGetPollingStatus.mockResolvedValue({ paused: true, nextPollAt: "" });
    mockGetFilters.mockResolvedValue([
      {
        ID: "f1",
        Name: "Backend",
        Keywords: "go",
        Location: "London",
        Source: "reed_uk",
        Enabled: true,
        CreatedAt: "2026-02-08T10:00:00Z",
        UpdatedAt: "2026-02-08T10:00:00Z",
      },
    ]);
    mockPollNow.mockResolvedValue({
      runID: "run-fail",
      startedAt: "2026-02-14T12:00:00Z",
      completedAt: "2026-02-14T12:00:01Z",
      durationMs: 1000,
      totalFilters: 0,
      failedFilters: 0,
      newJobs: 0,
      skipped: 0,
      cycleError: "listing enabled filters: database unavailable",
      diagnosticsPath: "/tmp/poll-run-run-fail.json",
      diagnosticsError: "",
      filters: [],
    });

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByRole("button", { name: "Poll Now" }));

    await waitFor(() => {
      expect(screen.getByText("Poll failed")).toBeInTheDocument();
      expect(screen.getByText(/cycle error:/i)).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: /save report/i })).toBeInTheDocument();
  });

  it("does not show poll-complete toast when poll returns empty results", async () => {
    mockGetPollingStatus.mockResolvedValue({ paused: true, nextPollAt: "" });
    mockGetFilters.mockResolvedValue([
      {
        ID: "f1",
        Name: "Backend",
        Keywords: "go",
        Location: "London",
        Source: "reed_uk",
        Enabled: true,
        CreatedAt: "2026-02-08T10:00:00Z",
        UpdatedAt: "2026-02-08T10:00:00Z",
      },
    ]);
    mockPollNow.mockResolvedValue(noOpRun);

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByRole("button", { name: "Poll Now" }));

    await waitFor(() => {
      expect(mockPollNow).toHaveBeenCalledTimes(1);
    });
    expect(screen.queryByText(/Poll complete:/)).not.toBeInTheDocument();
  });

  it("opens settings panel when gear icon is clicked", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });

    await userEvent.click(
      screen.getByRole("button", { name: /open settings/i })
    );

    expect(
      screen.getByRole("dialog", { name: "Settings" })
    ).toBeInTheDocument();
  });

  it("closes settings panel when close button is clicked", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });

    await userEvent.click(
      screen.getByRole("button", { name: /open settings/i })
    );
    expect(
      screen.getByRole("dialog", { name: "Settings" })
    ).toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: /close settings/i })
    );
    expect(
      screen.queryByRole("dialog", { name: "Settings" })
    ).not.toBeInTheDocument();
  });

  it("closes settings panel when a job is selected", async () => {
    mockGetJobs.mockResolvedValue([fakeJob()]);
    mockGetJobCount.mockResolvedValue(1);

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("Go Developer")).toBeInTheDocument();
    });

    // Open settings.
    await userEvent.click(
      screen.getByRole("button", { name: /open settings/i })
    );
    expect(
      screen.getByRole("dialog", { name: "Settings" })
    ).toBeInTheDocument();

    // Click a job tile.
    await userEvent.click(screen.getByText("Go Developer"));

    // Settings should auto-close.
    expect(
      screen.queryByRole("dialog", { name: "Settings" })
    ).not.toBeInTheDocument();
  });

  // --- Keyboard Shortcut Integration ---

  it("Escape closes settings panel", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });

    // Open settings via click.
    await userEvent.click(
      screen.getByRole("button", { name: /open settings/i })
    );
    expect(
      screen.getByRole("dialog", { name: "Settings" })
    ).toBeInTheDocument();

    // Press Escape.
    act(() => {
      document.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape", bubbles: true })
      );
    });

    expect(
      screen.queryByRole("dialog", { name: "Settings" })
    ).not.toBeInTheDocument();
  });

  it(", opens settings panel", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });

    act(() => {
      document.dispatchEvent(
        new KeyboardEvent("keydown", { key: ",", bubbles: true })
      );
    });

    expect(
      screen.getByRole("dialog", { name: "Settings" })
    ).toBeInTheDocument();
  });

  it("/ focuses search input", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByLabelText("Search jobs")).toBeInTheDocument();
    });

    act(() => {
      document.dispatchEvent(
        new KeyboardEvent("keydown", { key: "/", bubbles: true })
      );
    });

    expect(document.activeElement).toBe(screen.getByLabelText("Search jobs"));
  });

  it("j selects first job in the list", async () => {
    mockGetJobs.mockResolvedValue([fakeJob("j1", "Go Dev"), fakeJob("j2", "React Dev")]);
    mockGetJobCount.mockResolvedValue(2);

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("Go Dev")).toBeInTheDocument();
    });
    // Ensure shortcuts are not suppressed by an input focus state.
    (document.activeElement as HTMLElement | null)?.blur();

    // Job detail should open for the first job.
    // Trigger inside waitFor to avoid races with async filtered-job-id wiring.
    await waitFor(() => {
      act(() => {
        document.dispatchEvent(
          new KeyboardEvent("keydown", { key: "j", bubbles: true })
        );
      });
      expect(
        screen.getByRole("button", { name: /close detail/i })
      ).toBeInTheDocument();
    });
  });

  it("? opens shortcuts help overlay", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });

    act(() => {
      document.dispatchEvent(
        new KeyboardEvent("keydown", { key: "?", bubbles: true })
      );
    });

    expect(
      screen.getByRole("dialog", { name: "Keyboard shortcuts" })
    ).toBeInTheDocument();
  });

  it("Escape closes job detail panel", async () => {
    mockGetJobs.mockResolvedValue([fakeJob()]);
    mockGetJobCount.mockResolvedValue(1);

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("Go Developer")).toBeInTheDocument();
    });

    // Select a job by clicking.
    await userEvent.click(screen.getByText("Go Developer"));
    expect(
      screen.getByRole("button", { name: /close detail/i })
    ).toBeInTheDocument();

    // Press Escape to close.
    act(() => {
      document.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape", bubbles: true })
      );
    });

    expect(
      screen.queryByRole("button", { name: /close detail/i })
    ).not.toBeInTheDocument();
  });
});
