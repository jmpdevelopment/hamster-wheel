import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { Button } from "./Button";

describe("Button", () => {
  it("renders children text", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByRole("button", { name: "Click me" })).toBeInTheDocument();
  });

  it("defaults to primary variant classes", () => {
    render(<Button>Primary</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("bg-hw-accent");
    expect(btn.className).toContain("text-hw-bg");
  });

  it("applies secondary variant classes", () => {
    render(<Button variant="secondary">Secondary</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("border-hw-border");
    expect(btn.className).toContain("text-hw-text-muted");
  });

  it("applies danger variant classes", () => {
    render(<Button variant="danger">Danger</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("bg-hw-danger");
    expect(btn.className).toContain("text-white");
  });

  it("applies ghost variant classes", () => {
    render(<Button variant="ghost">Ghost</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("text-hw-text-muted");
    expect(btn.className).not.toContain("bg-hw-accent");
    expect(btn.className).not.toContain("bg-hw-danger");
  });

  it("defaults to md size classes", () => {
    render(<Button>MD</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("px-3");
    expect(btn.className).toContain("text-sm");
  });

  it("applies sm size classes", () => {
    render(<Button size="sm">SM</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("px-2");
    expect(btn.className).toContain("text-xs");
  });

  it("has focus-visible ring classes", () => {
    render(<Button>Focus</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("focus-visible:ring-2");
    expect(btn.className).toContain("focus-visible:ring-hw-accent");
  });

  it("shows spinner and disables when loading", () => {
    const { container } = render(<Button loading>Loading</Button>);
    const btn = screen.getByRole("button");
    expect(btn).toBeDisabled();
    expect(container.querySelector("svg.animate-spin")).toBeInTheDocument();
    expect(screen.getByText("Loading")).toBeInTheDocument();
  });

  it("does not show spinner when not loading", () => {
    const { container } = render(<Button>Normal</Button>);
    expect(container.querySelector("svg.animate-spin")).not.toBeInTheDocument();
  });

  it("disables button when disabled prop is set", () => {
    render(<Button disabled>Disabled</Button>);
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("calls onClick handler", async () => {
    const onClick = vi.fn();
    render(<Button onClick={onClick}>Click</Button>);
    await userEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("does not call onClick when disabled", async () => {
    const onClick = vi.fn();
    render(<Button disabled onClick={onClick}>Click</Button>);
    await userEvent.click(screen.getByRole("button"));
    expect(onClick).not.toHaveBeenCalled();
  });

  it("does not call onClick when loading", async () => {
    const onClick = vi.fn();
    render(<Button loading onClick={onClick}>Click</Button>);
    await userEvent.click(screen.getByRole("button"));
    expect(onClick).not.toHaveBeenCalled();
  });

  it("spreads additional HTML attributes", () => {
    render(<Button type="submit" data-testid="submit-btn">Submit</Button>);
    const btn = screen.getByTestId("submit-btn");
    expect(btn).toHaveAttribute("type", "submit");
  });

  it("appends custom className", () => {
    render(<Button className="mt-4">Styled</Button>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("mt-4");
  });
});
