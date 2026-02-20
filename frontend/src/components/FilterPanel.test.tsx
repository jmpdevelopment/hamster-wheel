import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { FilterPanel } from "./FilterPanel";

const fakeFilter = (id: string, name: string, enabled = true) => ({
  ID: `group-${id}`,
  Name: name,
  Keywords: "go developer",
  Location: "London",
  Sources: ["reed_uk"],
  FilterIDs: [id],
  Filters: [
    {
      ID: id,
      Name: name,
      Keywords: "go developer",
      Location: "London",
      Source: "reed_uk",
      Enabled: enabled,
      CreatedAt: "2026-02-08T10:00:00Z",
      UpdatedAt: "2026-02-08T10:00:00Z",
    },
  ],
  Enabled: enabled,
  AllEnabled: enabled,
  EnabledSourceCount: enabled ? 1 : 0,
});

const defaultProps = {
  filters: [fakeFilter("f1", "Backend"), fakeFilter("f2", "Frontend", false)],
  jobCountsByFilterId: { "group-f1": 12, "group-f2": 4 },
  loading: false,
  onCreateFilter: vi.fn().mockResolvedValue(undefined),
  onToggleFilter: vi.fn().mockResolvedValue(undefined),
  onDeleteFilter: vi.fn().mockResolvedValue(undefined) as (
    filter: ReturnType<typeof fakeFilter>,
    deleteAssociatedJobs: boolean
  ) => Promise<void>,
};

describe("FilterPanel", () => {
  it("renders filter list", () => {
    render(<FilterPanel {...defaultProps} />);

    expect(screen.getByText("Backend")).toBeInTheDocument();
    expect(screen.getByText("Frontend")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(<FilterPanel {...defaultProps} filters={[]} loading={true} />);
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("shows empty state when no filters", () => {
    render(<FilterPanel {...defaultProps} filters={[]} loading={false} />);
    expect(screen.getByText("No filters")).toBeInTheDocument();
  });

  it("shows create form when + button is clicked", async () => {
    render(<FilterPanel {...defaultProps} />);

    await userEvent.click(screen.getByRole("button", { name: /add filter/i }));
    expect(screen.getByLabelText("Filter name")).toBeInTheDocument();
  });

  it("hides + button while form is open", async () => {
    render(<FilterPanel {...defaultProps} />);

    await userEvent.click(screen.getByRole("button", { name: /add filter/i }));
    expect(screen.queryByRole("button", { name: /add filter/i })).not.toBeInTheDocument();
  });

  it("hides form on cancel", async () => {
    render(<FilterPanel {...defaultProps} />);

    await userEvent.click(screen.getByRole("button", { name: /add filter/i }));
    await userEvent.click(screen.getByRole("button", { name: /cancel/i }));

    expect(screen.queryByLabelText("Filter name")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /add filter/i })).toBeInTheDocument();
  });
});
