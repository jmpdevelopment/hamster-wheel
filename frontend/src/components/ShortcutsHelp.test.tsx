import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { ShortcutsHelp } from "./ShortcutsHelp";

describe("ShortcutsHelp", () => {
  it("renders dialog with keyboard shortcuts label", () => {
    render(<ShortcutsHelp onClose={vi.fn()} />);
    expect(
      screen.getByRole("dialog", { name: "Keyboard shortcuts" })
    ).toBeInTheDocument();
  });

  it("renders all shortcut entries", () => {
    render(<ShortcutsHelp onClose={vi.fn()} />);
    expect(screen.getByText("Next job")).toBeInTheDocument();
    expect(screen.getByText("Previous job")).toBeInTheDocument();
    expect(screen.getByText("Open job detail")).toBeInTheDocument();
    expect(screen.getByText("Focus search")).toBeInTheDocument();
    expect(screen.getByText("Poll now")).toBeInTheDocument();
    expect(screen.getByText("Open settings")).toBeInTheDocument();
    expect(screen.getByText("Show shortcuts")).toBeInTheDocument();
    expect(screen.getByText("Close / cancel")).toBeInTheDocument();
    expect(screen.getByText("Delete job (press twice)")).toBeInTheDocument();
  });

  it("calls onClose when backdrop is clicked", async () => {
    const onClose = vi.fn();
    render(<ShortcutsHelp onClose={onClose} />);
    await userEvent.click(
      screen.getByRole("dialog", { name: "Keyboard shortcuts" })
    );
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not call onClose when content is clicked", async () => {
    const onClose = vi.fn();
    render(<ShortcutsHelp onClose={onClose} />);
    await userEvent.click(screen.getByText("Keyboard Shortcuts"));
    expect(onClose).not.toHaveBeenCalled();
  });
});
