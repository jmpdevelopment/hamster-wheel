import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { SearchInput } from "./SearchInput";

describe("SearchInput", () => {
  it("renders with placeholder", () => {
    render(<SearchInput value="" onChange={() => {}} />);
    expect(screen.getByPlaceholderText("Search jobs...")).toBeInTheDocument();
  });

  it("has aria-label for accessibility", () => {
    render(<SearchInput value="" onChange={() => {}} />);
    expect(screen.getByLabelText("Search jobs")).toBeInTheDocument();
  });

  it("renders search icon", () => {
    const { container } = render(<SearchInput value="" onChange={() => {}} />);
    const svg = container.querySelector("svg[aria-hidden]");
    expect(svg).toBeInTheDocument();
  });

  it("calls onChange when user types", async () => {
    const onChange = vi.fn();
    render(<SearchInput value="" onChange={onChange} />);

    await userEvent.type(screen.getByLabelText("Search jobs"), "go");
    expect(onChange).toHaveBeenCalledWith("g");
    expect(onChange).toHaveBeenCalledWith("o");
  });

  it("does not show clear button when value is empty", () => {
    render(<SearchInput value="" onChange={() => {}} />);
    expect(screen.queryByLabelText("Clear search")).not.toBeInTheDocument();
  });

  it("shows clear button when value is non-empty", () => {
    render(<SearchInput value="test" onChange={() => {}} />);
    expect(screen.getByLabelText("Clear search")).toBeInTheDocument();
  });

  it("calls onChange with empty string when clear button is clicked", async () => {
    const onChange = vi.fn();
    render(<SearchInput value="test" onChange={onChange} />);

    await userEvent.click(screen.getByLabelText("Clear search"));
    expect(onChange).toHaveBeenCalledWith("");
  });

  it("displays the current value", () => {
    render(<SearchInput value="hello" onChange={() => {}} />);
    expect(screen.getByLabelText("Search jobs")).toHaveValue("hello");
  });
});
