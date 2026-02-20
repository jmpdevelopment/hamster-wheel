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
    expect(screen.getByLabelText(/reed uk/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/adzuna uk/i)).toBeInTheDocument();
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

  it("submit button is disabled when no sources are selected", async () => {
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={vi.fn()} />);

    await userEvent.type(screen.getByLabelText("Filter name"), "My Filter");
    await userEvent.type(screen.getByLabelText("Keywords"), "react developer");
    await userEvent.click(screen.getByLabelText(/reed uk/i));

    expect(screen.getByRole("button", { name: /create/i })).toBeDisabled();
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
      ["reed_uk"]
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

  it("shows source list as reed_uk by default", () => {
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByText(/^Sources:\s*reed_uk$/)).toBeInTheDocument();
    expect(
      screen.queryByText(
        "Adzuna provides a description snippet, not the full job ad. For AI matching, we include the snippet and job URL in the prompt when available."
      )
    ).not.toBeInTheDocument();
  });

  it("submits selected adzuna + reed sources", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(<CreateFilterForm onSubmit={onSubmit} onCancel={vi.fn()} />);

    await userEvent.type(screen.getByLabelText("Filter name"), "Adzuna Filter");
    await userEvent.type(screen.getByLabelText("Keywords"), "go");
    await userEvent.click(screen.getByLabelText(/adzuna uk/i));
    await userEvent.click(screen.getByRole("button", { name: /create/i }));

    expect(onSubmit).toHaveBeenCalledWith(
      "Adzuna Filter",
      "go",
      "",
      ["reed_uk", "adzuna_gb"]
    );
  });

  it("shows snippet notice when adzuna source is selected", async () => {
    render(<CreateFilterForm onSubmit={vi.fn()} onCancel={vi.fn()} />);

    await userEvent.click(screen.getByLabelText(/adzuna uk/i));

    expect(
      screen.getByText(
        "Adzuna provides a description snippet, not the full job ad. For AI matching, we include the snippet and job URL in the prompt when available."
      )
    ).toBeInTheDocument();
  });
});
