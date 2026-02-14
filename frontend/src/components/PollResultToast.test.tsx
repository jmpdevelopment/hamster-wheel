import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, afterEach } from "vitest";
import { PollResultToast } from "./PollResultToast";

const fakeResults = [
  { FilterID: "f1", FilterName: "Backend London", Source: "reed_uk", NewJobs: 3, Skipped: 2, Err: null },
  { FilterID: "f2", FilterName: "Remote React", Source: "reed_uk", NewJobs: 1, Skipped: 0, Err: null },
];

afterEach(() => {
  vi.useRealTimers();
});

describe("PollResultToast", () => {
  it("renders nothing when results is null", () => {
    const { container } = render(
      <PollResultToast results={null} onDismiss={() => {}} />
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders nothing when results is empty", () => {
    const { container } = render(
      <PollResultToast results={[]} onDismiss={() => {}} />
    );
    expect(container.firstChild).toBeNull();
  });

  it("shows total new and skipped", () => {
    render(<PollResultToast results={fakeResults} onDismiss={() => {}} />);
    expect(screen.getByText("Poll complete: 4 new, 2 skipped")).toBeInTheDocument();
  });

  it("shows per-filter results", () => {
    render(<PollResultToast results={fakeResults} onDismiss={() => {}} />);
    expect(screen.getByText(/Backend London/)).toBeInTheDocument();
    expect(screen.getByText(/Remote React/)).toBeInTheDocument();
  });

  it("shows error for failed filter", () => {
    const withError = [
      { FilterID: "f1", FilterName: "Broken", Source: "reed_uk", NewJobs: 0, Skipped: 0, Err: "timeout" },
    ];
    render(<PollResultToast results={withError} onDismiss={() => {}} />);
    expect(
      screen.getByText("Poll complete: 0 new, 0 skipped, 1 failed")
    ).toBeInTheDocument();
    expect(screen.getByText("error: timeout")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("calls onDismiss when dismiss button is clicked", async () => {
    const onDismiss = vi.fn();
    render(<PollResultToast results={fakeResults} onDismiss={onDismiss} />);

    await userEvent.click(screen.getByRole("button", { name: /dismiss/i }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("auto-dismisses after 5 seconds", () => {
    vi.useFakeTimers();
    const onDismiss = vi.fn();
    render(<PollResultToast results={fakeResults} onDismiss={onDismiss} />);

    expect(onDismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(5000);
    expect(onDismiss).toHaveBeenCalledOnce();
  });
});
