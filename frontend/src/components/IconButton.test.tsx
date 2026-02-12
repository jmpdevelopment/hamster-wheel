import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { IconButton } from "./IconButton";

describe("IconButton", () => {
  it("renders children content", () => {
    render(<IconButton aria-label="Close">✕</IconButton>);
    expect(screen.getByRole("button")).toHaveTextContent("✕");
  });

  it("has correct aria-label", () => {
    render(<IconButton aria-label="Close panel">✕</IconButton>);
    expect(screen.getByRole("button", { name: "Close panel" })).toBeInTheDocument();
  });

  it("has focus-visible ring classes", () => {
    render(<IconButton aria-label="Close">✕</IconButton>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("focus-visible:ring-2");
    expect(btn.className).toContain("focus-visible:ring-hw-accent");
  });

  it("calls onClick handler", async () => {
    const onClick = vi.fn();
    render(<IconButton aria-label="Close" onClick={onClick}>✕</IconButton>);
    await userEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("disables button when disabled prop is set", () => {
    render(<IconButton aria-label="Close" disabled>✕</IconButton>);
    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("does not call onClick when disabled", async () => {
    const onClick = vi.fn();
    render(<IconButton aria-label="Close" disabled onClick={onClick}>✕</IconButton>);
    await userEvent.click(screen.getByRole("button"));
    expect(onClick).not.toHaveBeenCalled();
  });

  it("appends custom className", () => {
    render(<IconButton aria-label="Close" className="ml-2">✕</IconButton>);
    const btn = screen.getByRole("button");
    expect(btn.className).toContain("ml-2");
  });
});
