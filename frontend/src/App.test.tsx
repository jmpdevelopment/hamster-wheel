import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import App from "./App";

const fakeJob = {
  ID: "j1",
  Source: "reed_uk",
  SourceID: "src-1",
  Title: "Go Developer",
  Company: "Acme",
  Location: "London",
  Description: "A job",
  URL: "https://example.com",
  PostedAt: "2026-02-08T10:00:00Z",
  DiscoveredAt: "2026-02-08T11:00:00Z",
  FilterID: "f1",
};

const mockGetJobs = vi.fn();
const mockGetJobCount = vi.fn();

// Mock all Wails bindings.
vi.mock("../bindings/hamster-wheel/jobservice", () => ({
  GetJobs: (...args: unknown[]) => mockGetJobs(...args),
  GetJobCount: (...args: unknown[]) => mockGetJobCount(...args),
  DeleteJob: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("../bindings/hamster-wheel/filterservice", () => ({
  GetFilters: vi.fn().mockResolvedValue([]),
  CreateFilter: vi.fn().mockResolvedValue("f1"),
  UpdateFilter: vi.fn().mockResolvedValue(undefined),
  DeleteFilter: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("../bindings/hamster-wheel/pollingservice", () => ({
  PollNow: vi.fn().mockResolvedValue([]),
  GetPollingStatus: vi
    .fn()
    .mockResolvedValue({ paused: false, nextPollAt: "" }),
  SetPollingPaused: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("../bindings/hamster-wheel/settingsservice", () => ({
  GetReedAPIKey: vi.fn().mockResolvedValue(""),
  SetReedAPIKey: vi.fn().mockResolvedValue(undefined),
  GetTheme: vi.fn().mockResolvedValue(""),
  SetTheme: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@wailsio/runtime", () => ({
  Browser: { OpenURL: vi.fn() },
}));

vi.mock("react-virtualized-auto-sizer", () => ({
  AutoSizer: ({
    renderProp,
  }: {
    renderProp: (size: { height: number; width: number }) => React.ReactNode;
  }) => renderProp({ height: 600, width: 400 }),
}));

beforeEach(() => {
  mockGetJobs.mockResolvedValue([]);
  mockGetJobCount.mockResolvedValue(0);
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
    mockGetJobs.mockResolvedValue([fakeJob]);
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
});
