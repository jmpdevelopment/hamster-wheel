import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import App from "./App";

// Mock all Wails bindings.
vi.mock("../bindings/hamster-wheel/jobservice", () => ({
  GetJobs: vi.fn().mockResolvedValue([]),
  GetJobCount: vi.fn().mockResolvedValue(0),
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
}));

vi.mock("@wailsio/runtime", () => ({
  Browser: { OpenURL: vi.fn() },
}));

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
});
