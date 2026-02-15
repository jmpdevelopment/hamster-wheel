import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { Header } from "./Header";

const defaultProps = {
  onPollNow: vi.fn(),
  isPolling: false,
  pollingPaused: false,
  nextPollAt: "",
  hasFilters: true,
  hasEnabledFilters: true,
  onOpenSettings: vi.fn(),
};

describe("Header", () => {
  it("renders the app title", () => {
    render(<Header {...defaultProps} />);
    expect(screen.getByText("Hamster Wheel")).toBeInTheDocument();
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
    expect(
      screen.getByText("Polling...", { selector: "span" })
    ).toBeInTheDocument();
  });

  it("shows auto-polling disabled status when paused", () => {
    render(<Header {...defaultProps} pollingPaused={true} />);
    expect(screen.getByText("Auto Polling is Disabled.")).toBeInTheDocument();
  });

  it("shows next poll time when not paused", () => {
    render(
      <Header
        {...defaultProps}
        pollingPaused={false}
        nextPollAt="2026-02-08T22:00:00Z"
      />
    );
    expect(screen.getByText(/Next:/)).toBeInTheDocument();
  });

  it("shows polling status while polling and next poll time is not yet available", () => {
    render(
      <Header
        {...defaultProps}
        isPolling={true}
        nextPollAt=""
        pollingPaused={false}
      />
    );
    expect(
      screen.getByText("Polling...", { selector: "span" })
    ).toBeInTheDocument();
  });

  it("does not show polling status before polling starts", () => {
    render(<Header {...defaultProps} isPolling={false} nextPollAt="" pollingPaused={false} />);
    expect(screen.queryByText("Polling...")).not.toBeInTheDocument();
  });

  it("keeps Poll Now enabled while auto-polling is enabled", () => {
    render(<Header {...defaultProps} pollingPaused={false} />);
    expect(screen.getByRole("button", { name: /poll now/i })).toBeEnabled();
  });

  it("renders settings gear button", () => {
    render(<Header {...defaultProps} />);
    expect(
      screen.getByRole("button", { name: /open settings/i })
    ).toBeInTheDocument();
  });

  it("calls onOpenSettings when gear button is clicked", async () => {
    const onOpenSettings = vi.fn();
    render(<Header {...defaultProps} onOpenSettings={onOpenSettings} />);
    await userEvent.click(
      screen.getByRole("button", { name: /open settings/i })
    );
    expect(onOpenSettings).toHaveBeenCalledOnce();
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

    it("shows guidance to enable a filter", () => {
      render(<Header {...noEnabledProps} />);
      expect(
        screen.getByText("Enable a filter to start monitoring")
      ).toBeInTheDocument();
    });
  });
});
