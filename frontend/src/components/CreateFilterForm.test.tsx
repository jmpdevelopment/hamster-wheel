import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { CreateFilterForm } from "./CreateFilterForm";

describe("CreateFilterForm", () => {
  it("renders all form fields", () => {
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={vi.fn()} />);

    expect(screen.getByLabelText("Filter name")).toBeInTheDocument();
    expect(screen.getByLabelText("Keywords")).toBeInTheDocument();
    expect(screen.getByLabelText("Location")).toBeInTheDocument();
    expect(screen.getByLabelText("Source")).toBeInTheDocument();
  });

  it("submit button is disabled when name is empty", () => {
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={vi.fn()} />);

    const button = screen.getByRole("button", { name: /create/i });
    expect(button).toBeDisabled();
  });

  it("submit button is disabled when keywords is empty", async () => {
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={vi.fn()} />);

    await userEvent.type(screen.getByLabelText("Filter name"), "My Filter");
    const button = screen.getByRole("button", { name: /create/i });
    expect(button).toBeDisabled();
  });

  it("submit button enables when name and keywords are filled", async () => {
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={vi.fn()} />);

    await userEvent.type(screen.getByLabelText("Filter name"), "My Filter");
    await userEvent.type(screen.getByLabelText("Keywords"), "react developer");
    const button = screen.getByRole("button", { name: /create/i });
    expect(button).toBeEnabled();
  });

  it("calls onSubmit with trimmed values", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<CreateFilterForm onSubmit={onSubmit} onCancel={vi.fn()} />);

    await userEvent.type(screen.getByLabelText("Filter name"), "  My Filter  ");
    await userEvent.type(screen.getByLabelText("Keywords"), "  react  ");
    await userEvent.type(screen.getByLabelText("Location"), "  London  ");
    await userEvent.click(screen.getByRole("button", { name: /create/i }));

    expect(onSubmit).toHaveBeenCalledWith(
      "My Filter",
      "react",
      "London",
      "reed_uk"
    );
  });

  it("clears form after successful submit", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<CreateFilterForm onSubmit={onSubmit} onCancel={vi.fn()} />);

    await userEvent.type(screen.getByLabelText("Filter name"), "My Filter");
    await userEvent.type(screen.getByLabelText("Keywords"), "react");
    await userEvent.click(screen.getByRole("button", { name: /create/i }));

    expect(screen.getByLabelText("Filter name")).toHaveValue("");
    expect(screen.getByLabelText("Keywords")).toHaveValue("");
  });

  it("calls onCancel when cancel button is clicked", async () => {
    const onCancel = vi.fn();
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={onCancel} />);

    await userEvent.click(screen.getByRole("button", { name: /cancel/i }));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("shows source as reed_uk", () => {
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByText(/^Source:\s*reed_uk$/)).toBeInTheDocument();
    expect(
      screen.queryByText(
        "Adzuna provides a description snippet, not the full job ad."
      )
    ).not.toBeInTheDocument();
  });

  it("submits selected adzuna source", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<CreateFilterForm onSubmit={onSubmit} onCancel={vi.fn()} />);

    await userEvent.type(screen.getByLabelText("Filter name"), "Adzuna Filter");
    await userEvent.type(screen.getByLabelText("Keywords"), "go");
    await userEvent.selectOptions(screen.getByLabelText("Source"), "adzuna_gb");
    await userEvent.click(screen.getByRole("button", { name: /create/i }));

    expect(onSubmit).toHaveBeenCalledWith(
      "Adzuna Filter",
      "go",
      "",
      "adzuna_gb"
    );
  });

  it("shows snippet notice when adzuna source is selected", async () => {
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={vi.fn()} />);

    await userEvent.selectOptions(screen.getByLabelText("Source"), "adzuna_gb");

    expect(
      screen.getByText("Adzuna provides a description snippet, not the full job ad.")
    ).toBeInTheDocument();
  });
});
