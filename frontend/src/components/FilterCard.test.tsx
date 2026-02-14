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
      <FilterCard
        filter={fakeFilter()}
        associatedJobCount={3}
        onToggle={async () => {}}
        onDelete={async () => {}}
      />
    );

    expect(screen.getByText("Backend London")).toBeInTheDocument();
    expect(screen.getByText(/go developer/)).toBeInTheDocument();
    expect(screen.getByText("reed_uk")).toBeInTheDocument();
  });

  it("shows location with separator", () => {
    render(
      <FilterCard
        filter={fakeFilter()}
        associatedJobCount={3}
        onToggle={async () => {}}
        onDelete={async () => {}}
      />
    );

    expect(screen.getByText(/· London/)).toBeInTheDocument();
  });

  it("shows ON when enabled", () => {
    render(
      <FilterCard
        filter={fakeFilter({ Enabled: true })}
        associatedJobCount={0}
        onToggle={async () => {}}
        onDelete={async () => {}}
      />
    );

    expect(screen.getByText("ON")).toBeInTheDocument();
  });

  it("shows OFF when disabled", () => {
    render(
      <FilterCard
        filter={fakeFilter({ Enabled: false })}
        associatedJobCount={0}
        onToggle={async () => {}}
        onDelete={async () => {}}
      />
    );

    expect(screen.getByText("OFF")).toBeInTheDocument();
  });

  it("calls onToggle with opposite value when toggle is clicked", async () => {
    const onToggle = vi.fn().mockResolvedValue(undefined);
    render(
      <FilterCard
        filter={fakeFilter({ Enabled: true })}
        associatedJobCount={0}
        onToggle={onToggle}
        onDelete={async () => {}}
      />
    );

    await userEvent.click(screen.getByRole("button", { name: /disable/i }));
    expect(onToggle).toHaveBeenCalledWith(false);
  });

  it("shows confirm buttons when delete is clicked", async () => {
    render(
      <FilterCard
        filter={fakeFilter()}
        associatedJobCount={3}
        onToggle={async () => {}}
        onDelete={async () => {}}
      />
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
    expect(
      screen.getByRole("checkbox", { name: /also delete 3 associated jobs/i })
    ).toBeInTheDocument();
  });

  it("calls onDelete(false) when delete is confirmed without checkbox", async () => {
    const onDelete = vi.fn().mockResolvedValue(undefined);

    render(
      <FilterCard
        filter={fakeFilter()}
        associatedJobCount={3}
        onToggle={async () => {}}
        onDelete={onDelete}
      />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /delete filter/i })
    );
    await userEvent.click(
      screen.getByRole("button", { name: /confirm delete/i })
    );
    expect(onDelete).toHaveBeenCalledWith(false);
  });

  it("calls onDelete(true) when checkbox is checked before confirm", async () => {
    const onDelete = vi.fn().mockResolvedValue(undefined);

    render(
      <FilterCard
        filter={fakeFilter()}
        associatedJobCount={2}
        onToggle={async () => {}}
        onDelete={onDelete}
      />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /delete filter/i })
    );
    await userEvent.click(
      screen.getByRole("checkbox", { name: /also delete 2 associated jobs/i })
    );
    await userEvent.click(
      screen.getByRole("button", { name: /confirm delete/i })
    );

    expect(onDelete).toHaveBeenCalledWith(true);
  });

  it("does not show associated-job checkbox when count is zero", async () => {
    render(
      <FilterCard
        filter={fakeFilter()}
        associatedJobCount={0}
        onToggle={async () => {}}
        onDelete={async () => {}}
      />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /delete filter/i })
    );

    expect(
      screen.queryByRole("checkbox", { name: /also delete/i })
    ).not.toBeInTheDocument();
  });

  it("does not call onDelete when delete is cancelled", async () => {
    const onDelete = vi.fn().mockResolvedValue(undefined);

    render(
      <FilterCard
        filter={fakeFilter()}
        associatedJobCount={3}
        onToggle={async () => {}}
        onDelete={onDelete}
      />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /delete filter/i })
    );
    await userEvent.click(
      screen.getByRole("button", { name: /cancel delete/i })
    );

    expect(onDelete).not.toHaveBeenCalled();
    expect(
      screen.queryByRole("button", { name: /confirm delete/i })
    ).not.toBeInTheDocument();
  });
});
