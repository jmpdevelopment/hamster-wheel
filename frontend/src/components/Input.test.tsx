import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { Input } from "./Input";

describe("Input", () => {
  it("renders with placeholder", () => {
    render(<Input placeholder="Enter text" />);
    expect(screen.getByPlaceholderText("Enter text")).toBeInTheDocument();
  });

  it("defaults to md size classes", () => {
    render(<Input placeholder="test" />);
    const input = screen.getByPlaceholderText("test");
    expect(input.className).toContain("py-1.5");
    expect(input.className).toContain("text-sm");
  });

  it("applies sm size classes", () => {
    render(<Input size="sm" placeholder="test" />);
    const input = screen.getByPlaceholderText("test");
    expect(input.className).toContain("py-1");
    expect(input.className).toContain("text-xs");
  });

  it("has focus-visible ring classes", () => {
    render(<Input placeholder="test" />);
    const input = screen.getByPlaceholderText("test");
    expect(input.className).toContain("focus-visible:ring-2");
    expect(input.className).toContain("focus-visible:ring-hw-accent");
  });

  it("has theme-aware border and background classes", () => {
    render(<Input placeholder="test" />);
    const input = screen.getByPlaceholderText("test");
    expect(input.className).toContain("bg-hw-bg");
    expect(input.className).toContain("border-hw-border");
    expect(input.className).toContain("focus:border-hw-accent");
  });

  it("calls onChange handler", async () => {
    const onChange = vi.fn();
    render(<Input placeholder="test" onChange={onChange} />);
    await userEvent.type(screen.getByPlaceholderText("test"), "hello");
    expect(onChange).toHaveBeenCalled();
  });

  it("disables input when disabled prop is set", () => {
    render(<Input placeholder="test" disabled />);
    expect(screen.getByPlaceholderText("test")).toBeDisabled();
  });

  it("spreads additional HTML attributes", () => {
    render(<Input placeholder="test" type="password" data-testid="pw" />);
    const input = screen.getByTestId("pw");
    expect(input).toHaveAttribute("type", "password");
  });

  it("appends custom className", () => {
    render(<Input placeholder="test" className="max-w-xs" />);
    const input = screen.getByPlaceholderText("test");
    expect(input.className).toContain("max-w-xs");
  });
});
