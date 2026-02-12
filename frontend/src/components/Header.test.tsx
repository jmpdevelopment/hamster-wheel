import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { Header } from "./Header";

const defaultProps = {
  jobCount: 0,
  onPollNow: vi.fn(),
  isPolling: false,
  pollingPaused: false,
  nextPollAt: "",
  onTogglePolling: vi.fn(),
  hasFilters: true,
  hasEnabledFilters: true,
};

describe("Header", () => {
  it("renders the app title", () => {
    render(<Header {...defaultProps} />);
    expect(screen.getByText("Hamster Wheel")).toBeInTheDocument();
  });

  it("shows the job count", () => {
    render(<Header {...defaultProps} jobCount={42} />);
    expect(screen.getByText("42 jobs")).toBeInTheDocument();
  });

  it("shows singular 'job' for count of 1", () => {
    render(<Header {...defaultProps} jobCount={1} />);
    expect(screen.getByText("1 job")).toBeInTheDocument();
  });

  it("calls onPollNow when button is clicked", async () => {
    const onPollNow = vi.fn();
    render(<Header {...defaultProps} onPollNow={onPollNow} />);

    await userEvent.click(screen.getByRole("button", { name: /poll now/i }));
    expect(onPollNow).toHaveBeenCalledOnce();
  });

  it("disables button while polling", () => {
    render(<Header {...defaultProps} isPolling={true} />);
    const button = screen.getByRole("button", { name: /polling.../i });
    expect(button).toBeDisabled();
  });

  it("shows 'Polling...' text while polling", () => {
    render(<Header {...defaultProps} isPolling={true} />);
    expect(screen.getByText("Polling...")).toBeInTheDocument();
  });

  it("shows Pause button when polling is active", () => {
    render(<Header {...defaultProps} />);
    expect(
      screen.getByRole("button", { name: /pause auto-polling/i })
    ).toBeInTheDocument();
  });

  it("shows Resume button when polling is paused", () => {
    render(<Header {...defaultProps} pollingPaused={true} />);
    expect(
      screen.getByRole("button", { name: /resume auto-polling/i })
    ).toBeInTheDocument();
  });

  it("shows 'Auto-poll paused' when paused", () => {
    render(<Header {...defaultProps} pollingPaused={true} />);
    expect(screen.getByText("Auto-poll paused")).toBeInTheDocument();
  });

  it("shows next poll time when not paused", () => {
    render(
      <Header {...defaultProps} nextPollAt="2026-02-08T22:00:00Z" />
    );
    expect(screen.getByText(/Next:/)).toBeInTheDocument();
  });

  it("calls onTogglePolling when Pause is clicked", async () => {
    const onTogglePolling = vi.fn();
    render(<Header {...defaultProps} onTogglePolling={onTogglePolling} />);

    await userEvent.click(
      screen.getByRole("button", { name: /pause auto-polling/i })
    );
    expect(onTogglePolling).toHaveBeenCalledOnce();
  });

  describe("when no filters exist", () => {
    const noFilterProps = {
      ...defaultProps,
      hasFilters: false,
      hasEnabledFilters: false,
    };

    it("disables Poll Now button", () => {
      render(<Header {...noFilterProps} />);
      expect(screen.getByRole("button", { name: /poll now/i })).toBeDisabled();
    });

    it("disables Pause/Resume button", () => {
      render(<Header {...noFilterProps} />);
      expect(
        screen.getByRole("button", { name: /no filters configured/i })
      ).toBeDisabled();
    });

    it("shows guidance to add a filter", () => {
      render(<Header {...noFilterProps} />);
      expect(
        screen.getByText("Add a filter to start monitoring")
      ).toBeInTheDocument();
    });

    it("does not call onPollNow when clicked", async () => {
      const onPollNow = vi.fn();
      render(<Header {...noFilterProps} onPollNow={onPollNow} />);

      const button = screen.getByRole("button", { name: /poll now/i });
      await userEvent.click(button);
      expect(onPollNow).not.toHaveBeenCalled();
    });
  });

  describe("when filters exist but none enabled", () => {
    const noEnabledProps = {
      ...defaultProps,
      hasFilters: true,
      hasEnabledFilters: false,
    };

    it("disables Poll Now button", () => {
      render(<Header {...noEnabledProps} />);
      expect(screen.getByRole("button", { name: /poll now/i })).toBeDisabled();
    });

    it("disables Pause/Resume button", () => {
      render(<Header {...noEnabledProps} />);
      expect(
        screen.getByRole("button", { name: /no filters enabled/i })
      ).toBeDisabled();
    });

    it("shows guidance to enable a filter", () => {
      render(<Header {...noEnabledProps} />);
      expect(
        screen.getByText("Enable a filter to start monitoring")
      ).toBeInTheDocument();
    });
  });
});
