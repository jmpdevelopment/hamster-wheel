import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { JobList } from "./JobList";

const fakeJob = (id: string, title: string, filterId = "f1") => ({
  ID: id,
  Source: "reed_uk",
  SourceID: `src-${id}`,
  Title: title,
  Company: "Acme",
  Location: "London",
  Description: "Desc",
  URL: "https://example.com",
  PostedAt: "2026-02-08T10:00:00Z",
  DiscoveredAt: "2026-02-08T11:00:00Z",
  FilterID: filterId,
});

const fakeFilter = (id: string, name: string) => ({
  ID: id,
  Name: name,
  Keywords: "kw",
  Location: "loc",
  Source: "reed_uk",
  Enabled: true,
  CreatedAt: "2026-02-08T10:00:00Z",
  UpdatedAt: "2026-02-08T10:00:00Z",
});

const defaultProps = {
  jobs: [fakeJob("j1", "Go Dev"), fakeJob("j2", "React Dev", "f2")],
  filters: [fakeFilter("f1", "Backend"), fakeFilter("f2", "Frontend")],
  loading: false,
  selectedJobId: null,
  onSelectJob: vi.fn(),
  filterByFilterId: null,
  onFilterChange: vi.fn(),
};

describe("JobList", () => {
  it("renders all jobs", () => {
    render(<JobList {...defaultProps} />);
    expect(screen.getByText("Go Dev")).toBeInTheDocument();
    expect(screen.getByText("React Dev")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(<JobList {...defaultProps} jobs={[]} loading={true} />);
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("shows empty state when no jobs", () => {
    render(<JobList {...defaultProps} jobs={[]} loading={false} />);
    expect(screen.getByText("No jobs yet")).toBeInTheDocument();
  });

  it("filters jobs by selected filter", () => {
    render(<JobList {...defaultProps} filterByFilterId="f1" />);
    expect(screen.getByText("Go Dev")).toBeInTheDocument();
    expect(screen.queryByText("React Dev")).not.toBeInTheDocument();
  });

  it("shows filter dropdown with filter names", () => {
    render(<JobList {...defaultProps} />);
    const dropdown = screen.getByRole("combobox", {
      name: /filter jobs/i,
    });
    expect(dropdown).toBeInTheDocument();
    expect(screen.getByText("Backend")).toBeInTheDocument();
    expect(screen.getByText("Frontend")).toBeInTheDocument();
  });

  it("calls onFilterChange when dropdown value changes", async () => {
    const onFilterChange = vi.fn();
    render(<JobList {...defaultProps} onFilterChange={onFilterChange} />);

    const dropdown = screen.getByRole("combobox", {
      name: /filter jobs/i,
    });
    await userEvent.selectOptions(dropdown, "f1");
    expect(onFilterChange).toHaveBeenCalledWith("f1");
  });

  it("calls onFilterChange with null when 'All Filters' is selected", async () => {
    const onFilterChange = vi.fn();
    render(
      <JobList
        {...defaultProps}
        filterByFilterId="f1"
        onFilterChange={onFilterChange}
      />
    );

    const dropdown = screen.getByRole("combobox", {
      name: /filter jobs/i,
    });
    await userEvent.selectOptions(dropdown, "");
    expect(onFilterChange).toHaveBeenCalledWith(null);
  });

  it("calls onSelectJob when a job card is clicked", async () => {
    const onSelectJob = vi.fn();
    render(<JobList {...defaultProps} onSelectJob={onSelectJob} />);

    await userEvent.click(screen.getByText("Go Dev"));
    expect(onSelectJob).toHaveBeenCalledWith("j1");
  });
});
