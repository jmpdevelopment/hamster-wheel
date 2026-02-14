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
});

const mockGetJobs = vi.fn();
const mockGetJobCount = vi.fn();
const mockDeleteJob = vi.fn();
const mockGetFilters = vi.fn();
const mockCreateFilter = vi.fn();
const mockUpdateFilter = vi.fn();
const mockDeleteFilter = vi.fn();
const mockPollNow = vi.fn();
const mockGetPollingStatus = vi.fn();
const mockSetPollingPaused = vi.fn();
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
}));

vi.mock("../bindings/hamster-wheel/settingsservice", () => ({
  HasReedAPIKey: vi.fn().mockResolvedValue(false),
  SetReedAPIKey: vi.fn().mockResolvedValue(undefined),
  ClearReedAPIKey: vi.fn().mockResolvedValue(undefined),
  GetTheme: vi.fn().mockResolvedValue(""),
  SetTheme: vi.fn().mockResolvedValue(undefined),
  GetKeyboardShortcuts: vi.fn().mockResolvedValue(""),
  SetKeyboardShortcuts: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@wailsio/runtime", () => ({
  Browser: { OpenURL: vi.fn() },
  Dialogs: { SaveFile: vi.fn().mockResolvedValue("") },
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
  mockGetJobs.mockResolvedValue([]);
  mockGetJobCount.mockResolvedValue(0);
  mockDeleteJob.mockResolvedValue(undefined);
  mockGetFilters.mockResolvedValue([]);
  mockCreateFilter.mockResolvedValue("f1");
  mockUpdateFilter.mockResolvedValue(undefined);
  mockDeleteFilter.mockResolvedValue(undefined);
  mockPollNow.mockResolvedValue(noOpRun);
  mockGetPollingStatus.mockResolvedValue({ paused: false, nextPollAt: "" });
  mockSetPollingPaused.mockResolvedValue(undefined);
});

describe("App", () => {
  it("renders the app title and layout", async () => {
    render(<App />);

    expect(screen.getByText("Hamster Wheel")).toBeInTheDocument();
    expect(screen.getByText("Poll Now")).toBeInTheDocument();

    // Wait for hooks to finish loading.
    await waitFor(() => {
      expect(screen.getByText("0 jobs")).toBeInTheDocument();
    });
  });

  it("shows empty states when no data", async () => {
    render(<App />);

    await waitFor(() => {
      expect(screen.getByText("No filters")).toBeInTheDocument();
      expect(screen.getByText("No jobs yet")).toBeInTheDocument();
    });
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
    await userEvent.click(screen.getByRole("button", { name: /^delete$/i }));
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
      expect(screen.getByText("0 jobs")).toBeInTheDocument();
    });

    await userEvent.click(screen.getByRole("button", { name: "Poll Now" }));

    await waitFor(() => {
      expect(screen.getByText("Poll failed")).toBeInTheDocument();
      expect(screen.getByText(/cycle error:/i)).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: /save report/i })).toBeInTheDocument();
  });

  it("does not show poll-complete toast when poll returns empty results", async () => {
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
      expect(screen.getByText("0 jobs")).toBeInTheDocument();
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
      expect(screen.getByText("0 jobs")).toBeInTheDocument();
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
      expect(screen.getByText("0 jobs")).toBeInTheDocument();
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
      expect(screen.getByText("0 jobs")).toBeInTheDocument();
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
      expect(screen.getByText("0 jobs")).toBeInTheDocument();
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
      expect(screen.getByText("0 jobs")).toBeInTheDocument();
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
