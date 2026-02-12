import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, afterEach } from "vitest";
import { Toast } from "./Toast";

afterEach(() => {
  vi.useRealTimers();
});

describe("Toast", () => {
  it("renders title text", () => {
    render(<Toast title="Operation complete" onDismiss={() => {}} />);
    expect(screen.getByText("Operation complete")).toBeInTheDocument();
  });

  it("renders children body content", () => {
    render(
      <Toast title="Done" onDismiss={() => {}}>
        <p>Details here</p>
      </Toast>
    );
    expect(screen.getByText("Details here")).toBeInTheDocument();
  });

  it("renders without children", () => {
    const { container } = render(
      <Toast title="Title only" onDismiss={() => {}} />
    );
    expect(container.firstChild).not.toBeNull();
  });

  it("defaults to info variant with role='status'", () => {
    render(<Toast title="Info" onDismiss={() => {}} />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("uses role='status' for success variant", () => {
    render(<Toast title="Success" variant="success" onDismiss={() => {}} />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("uses role='alert' for error variant", () => {
    render(<Toast title="Error" variant="error" onDismiss={() => {}} />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("renders info icon with accent color", () => {
    render(<Toast title="Info" variant="info" onDismiss={() => {}} />);
    const icon = screen.getByRole("status").querySelector("svg");
    expect(icon).not.toBeNull();
    expect(icon!.closest("span")!.className).toContain("text-hw-accent");
  });

  it("renders success icon with success color", () => {
    render(<Toast title="Success" variant="success" onDismiss={() => {}} />);
    const icon = screen.getByRole("status").querySelector("svg");
    expect(icon).not.toBeNull();
    expect(icon!.closest("span")!.className).toContain("text-hw-success");
  });

  it("renders error icon with danger color", () => {
    render(<Toast title="Error" variant="error" onDismiss={() => {}} />);
    const icon = screen.getByRole("alert").querySelector("svg");
    expect(icon).not.toBeNull();
    expect(icon!.closest("span")!.className).toContain("text-hw-danger");
  });

  it("has distinct icon shapes per variant", () => {
    const { unmount: u1 } = render(
      <Toast title="Info" variant="info" onDismiss={() => {}} />
    );
    const infoSvg = screen.getByRole("status").querySelector("svg")!;
    expect(infoSvg.querySelector("circle")).not.toBeNull();
    expect(infoSvg.querySelector("line")).not.toBeNull();
    u1();

    const { unmount: u2 } = render(
      <Toast title="Success" variant="success" onDismiss={() => {}} />
    );
    const successSvg = screen.getByRole("status").querySelector("svg")!;
    expect(successSvg.querySelector("path")).not.toBeNull();
    u2();

    render(<Toast title="Error" variant="error" onDismiss={() => {}} />);
    const errorSvg = screen.getByRole("alert").querySelector("svg")!;
    const errorPath = errorSvg.querySelector("path");
    expect(errorPath).not.toBeNull();
    // Error uses a triangle path (different from success checkmark)
    expect(errorPath!.getAttribute("d")).toContain("L");
  });

  it("icons are hidden from screen readers", () => {
    render(<Toast title="Test" onDismiss={() => {}} />);
    const svg = screen.getByRole("status").querySelector("svg");
    expect(svg!.getAttribute("aria-hidden")).toBe("true");
  });

  it("calls onDismiss when dismiss button is clicked", async () => {
    const onDismiss = vi.fn();
    render(<Toast title="Test" onDismiss={onDismiss} />);

    await userEvent.click(screen.getByRole("button", { name: /dismiss/i }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("auto-dismisses after default duration (5000ms)", () => {
    vi.useFakeTimers();
    const onDismiss = vi.fn();
    render(<Toast title="Test" onDismiss={onDismiss} />);

    expect(onDismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(5000);
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("auto-dismisses after custom duration", () => {
    vi.useFakeTimers();
    const onDismiss = vi.fn();
    render(<Toast title="Test" duration={2000} onDismiss={onDismiss} />);

    vi.advanceTimersByTime(1999);
    expect(onDismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it("does not auto-dismiss when duration is 0", () => {
    vi.useFakeTimers();
    const onDismiss = vi.fn();
    render(<Toast title="Test" duration={0} onDismiss={onDismiss} />);

    vi.advanceTimersByTime(10000);
    expect(onDismiss).not.toHaveBeenCalled();
  });

  it("has fixed positioning classes", () => {
    render(<Toast title="Test" onDismiss={() => {}} />);
    const container = screen.getByRole("status");
    expect(container.className).toContain("fixed");
    expect(container.className).toContain("bottom-4");
    expect(container.className).toContain("right-4");
    expect(container.className).toContain("z-50");
  });

  it("has hw-surface background", () => {
    render(<Toast title="Test" onDismiss={() => {}} />);
    expect(screen.getByRole("status").className).toContain("bg-hw-surface");
  });

  it("error variant has danger border", () => {
    render(<Toast title="Error" variant="error" onDismiss={() => {}} />);
    expect(screen.getByRole("alert").className).toContain("border-hw-danger");
  });

  it("success variant has success border", () => {
    render(<Toast title="Success" variant="success" onDismiss={() => {}} />);
    expect(screen.getByRole("status").className).toContain("border-hw-success");
  });
});
