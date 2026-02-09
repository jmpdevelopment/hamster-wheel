import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { FilterCard } from "./FilterCard";

const fakeFilter = (overrides = {}) => ({
  ID: "f1",
  Name: "Backend London",
  Keywords: "go developer",
  Location: "London",
  Source: "reed_uk",
  Enabled: true,
  CreatedAt: "2026-02-08T10:00:00Z",
  UpdatedAt: "2026-02-08T10:00:00Z",
  ...overrides,
});

describe("FilterCard", () => {
  it("renders filter name and details", () => {
    render(
      <FilterCard filter={fakeFilter()} onToggle={() => {}} onDelete={() => {}} />
    );

    expect(screen.getByText("Backend London")).toBeInTheDocument();
    expect(screen.getByText(/go developer/)).toBeInTheDocument();
    expect(screen.getByText("reed_uk")).toBeInTheDocument();
  });

  it("shows location with separator", () => {
    render(
      <FilterCard filter={fakeFilter()} onToggle={() => {}} onDelete={() => {}} />
    );

    // Location appears as "· London" in the details line.
    expect(screen.getByText(/· London/)).toBeInTheDocument();
  });

  it("shows ON when enabled", () => {
    render(
      <FilterCard
        filter={fakeFilter({ Enabled: true })}
        onToggle={() => {}}
        onDelete={() => {}}
      />
    );

    expect(screen.getByText("ON")).toBeInTheDocument();
  });

  it("shows OFF when disabled", () => {
    render(
      <FilterCard
        filter={fakeFilter({ Enabled: false })}
        onToggle={() => {}}
        onDelete={() => {}}
      />
    );

    expect(screen.getByText("OFF")).toBeInTheDocument();
  });

  it("calls onToggle with opposite value when toggle is clicked", async () => {
    const onToggle = vi.fn();
    render(
      <FilterCard
        filter={fakeFilter({ Enabled: true })}
        onToggle={onToggle}
        onDelete={() => {}}
      />
    );

    await userEvent.click(screen.getByRole("button", { name: /disable/i }));
    expect(onToggle).toHaveBeenCalledWith(false);
  });

  it("shows confirm buttons when delete is clicked", async () => {
    render(
      <FilterCard filter={fakeFilter()} onToggle={() => {}} onDelete={() => {}} />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /delete filter/i })
    );

    expect(
      screen.getByRole("button", { name: /confirm delete/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /cancel delete/i })
    ).toBeInTheDocument();
  });

  it("calls onDelete when delete is confirmed", async () => {
    const onDelete = vi.fn();

    render(
      <FilterCard filter={fakeFilter()} onToggle={() => {}} onDelete={onDelete} />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /delete filter/i })
    );
    await userEvent.click(
      screen.getByRole("button", { name: /confirm delete/i })
    );
    expect(onDelete).toHaveBeenCalledOnce();
  });

  it("does not call onDelete when delete is cancelled", async () => {
    const onDelete = vi.fn();

    render(
      <FilterCard filter={fakeFilter()} onToggle={() => {}} onDelete={onDelete} />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /delete filter/i })
    );
    await userEvent.click(
      screen.getByRole("button", { name: /cancel delete/i })
    );
    expect(onDelete).not.toHaveBeenCalled();
    // Confirm buttons should be hidden again.
    expect(
      screen.queryByRole("button", { name: /confirm delete/i })
    ).not.toBeInTheDocument();
  });
});
