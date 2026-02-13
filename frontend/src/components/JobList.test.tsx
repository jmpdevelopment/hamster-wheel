import { render, screen, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { JobList } from "./JobList";

vi.mock("react-virtualized-auto-sizer", () => ({
  AutoSizer: ({
    renderProp,
  }: {
    renderProp: (size: { height: number; width: number }) => React.ReactNode;
  }) => renderProp({ height: 600, width: 400 }),
}));

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
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });
  afterEach(() => {
    vi.useRealTimers();
  });

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
    expect(
      screen.getByText("Make sure you have enabled filters and try polling.")
    ).toBeInTheDocument();
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

  // Search tests

  it("renders the search input", () => {
    render(<JobList {...defaultProps} />);
    expect(screen.getByLabelText("Search jobs")).toBeInTheDocument();
  });

  it("filters jobs by search term after debounce", async () => {
    render(<JobList {...defaultProps} />);

    const searchInput = screen.getByLabelText("Search jobs");
    await userEvent.type(searchInput, "react");

    // Before debounce — both jobs visible
    expect(screen.getByText("Go Dev")).toBeInTheDocument();
    expect(screen.getByText("React Dev")).toBeInTheDocument();

    // After debounce
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(screen.queryByText("Go Dev")).not.toBeInTheDocument();
    expect(screen.getByText("React Dev")).toBeInTheDocument();
  });

  it("clears search and restores all jobs", async () => {
    render(<JobList {...defaultProps} />);

    const searchInput = screen.getByLabelText("Search jobs");
    await userEvent.type(searchInput, "react");
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(screen.queryByText("Go Dev")).not.toBeInTheDocument();

    await userEvent.click(screen.getByLabelText("Clear search"));
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(screen.getByText("Go Dev")).toBeInTheDocument();
    expect(screen.getByText("React Dev")).toBeInTheDocument();
  });

  it("shows 'No matching jobs' when search yields no results", async () => {
    render(<JobList {...defaultProps} />);

    const searchInput = screen.getByLabelText("Search jobs");
    await userEvent.type(searchInput, "xyznotfound");
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(screen.getByText("No matching jobs")).toBeInTheDocument();
    expect(
      screen.getByText("Try adjusting your search or filter.")
    ).toBeInTheDocument();
  });

  it("shows 'No matching jobs' when filter yields no results", () => {
    render(<JobList {...defaultProps} filterByFilterId="nonexistent" />);
    expect(screen.getByText("No matching jobs")).toBeInTheDocument();
  });

  it("multi-term search requires all terms to match", async () => {
    const jobs = [
      fakeJob("j1", "Senior Go Developer"),
      fakeJob("j2", "Junior Go Engineer"),
      fakeJob("j3", "React Developer"),
    ];
    render(<JobList {...defaultProps} jobs={jobs} />);

    const searchInput = screen.getByLabelText("Search jobs");
    await userEvent.type(searchInput, "go dev");
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(screen.getByText("Senior Go Developer")).toBeInTheDocument();
    expect(screen.queryByText("Junior Go Engineer")).not.toBeInTheDocument();
    expect(screen.queryByText("React Developer")).not.toBeInTheDocument();
  });

  it("combines search and filter", async () => {
    const jobs = [
      fakeJob("j1", "Go Dev", "f1"),
      fakeJob("j2", "React Dev", "f2"),
      fakeJob("j3", "Go Engineer", "f2"),
    ];
    render(<JobList {...defaultProps} jobs={jobs} filterByFilterId="f2" />);

    const searchInput = screen.getByLabelText("Search jobs");
    await userEvent.type(searchInput, "go");
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(screen.queryByText("Go Dev")).not.toBeInTheDocument();
    expect(screen.queryByText("React Dev")).not.toBeInTheDocument();
    expect(screen.getByText("Go Engineer")).toBeInTheDocument();
  });

  // Job count indicator tests

  it("shows total job count when no filter or search", () => {
    render(<JobList {...defaultProps} />);
    expect(screen.getByText("2 jobs")).toBeInTheDocument();
  });

  it("shows filtered count when search is active", async () => {
    render(<JobList {...defaultProps} />);

    const searchInput = screen.getByLabelText("Search jobs");
    await userEvent.type(searchInput, "react");
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(screen.getByText("1 of 2 jobs")).toBeInTheDocument();
  });

  it("shows filtered count when filter is active", () => {
    render(<JobList {...defaultProps} filterByFilterId="f1" />);
    expect(screen.getByText("1 of 2 jobs")).toBeInTheDocument();
  });

  it("does not show count when there are no jobs", () => {
    render(<JobList {...defaultProps} jobs={[]} />);
    expect(screen.queryByText(/^\d+ jobs$/)).not.toBeInTheDocument();
  });
});
