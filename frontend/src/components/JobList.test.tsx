import { render, screen, act, fireEvent } from "@testing-library/react";
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
  IsFavorite: false,
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
  onFilteredJobsChange: vi.fn(),
  onSetFavoriteJobs: vi.fn(),
  onToggleFavoriteJob: vi.fn(),
  onDeleteJobs: vi.fn().mockResolvedValue(undefined),
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

  it("shows Adzuna attribution link when Adzuna jobs are visible", () => {
    render(
      <JobList
        {...defaultProps}
        jobs={[{ ...fakeJob("j1", "Go Dev"), Source: "adzuna_gb" }]}
      />
    );

    const link = screen.getByRole("link", { name: "Jobs by Adzuna" });
    expect(link).toHaveAttribute("href", "https://www.adzuna.co.uk");
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

  it("calls onFilterChange when dropdown value changes", async () => {
    const onFilterChange = vi.fn();
    render(<JobList {...defaultProps} onFilterChange={onFilterChange} />);

    const dropdown = screen.getByRole("combobox", {
      name: /filter jobs/i,
    });
    await userEvent.selectOptions(dropdown, "f1");
    expect(onFilterChange).toHaveBeenCalledWith("f1");
  });

  it("calls onSelectJob when a job card is clicked", async () => {
    const onSelectJob = vi.fn();
    render(<JobList {...defaultProps} onSelectJob={onSelectJob} />);

    await userEvent.click(screen.getByText("Go Dev"));
    expect(onSelectJob).toHaveBeenCalledWith("j1");
  });

  it("filters jobs by search term after debounce", async () => {
    render(<JobList {...defaultProps} />);

    const searchInput = screen.getByLabelText("Search jobs");
    await userEvent.type(searchInput, "react");

    expect(screen.getByText("Go Dev")).toBeInTheDocument();
    expect(screen.getByText("React Dev")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(screen.queryByText("Go Dev")).not.toBeInTheDocument();
    expect(screen.getByText("React Dev")).toBeInTheDocument();
  });

  it("shows total job count when no filter or search", () => {
    render(<JobList {...defaultProps} />);
    expect(screen.getByText("2 jobs")).toBeInTheDocument();
  });

  it("calls onFilteredJobsChange with all visible IDs initially", () => {
    const onFilteredJobsChange = vi.fn();
    render(
      <JobList {...defaultProps} onFilteredJobsChange={onFilteredJobsChange} />
    );

    expect(onFilteredJobsChange).toHaveBeenCalledWith(["j1", "j2"]);
  });

  it("favorites selected jobs via bulk action", async () => {
    const onSetFavoriteJobs = vi.fn();
    render(<JobList {...defaultProps} onSetFavoriteJobs={onSetFavoriteJobs} />);

    await userEvent.click(
      screen.getByRole("checkbox", { name: /select job go dev/i })
    );
    await userEvent.click(screen.getByRole("button", { name: /^favorite$/i }));

    expect(onSetFavoriteJobs).toHaveBeenCalledWith(["j1"], true);
  });

  it("select-all chooses visible jobs and bulk delete calls onDeleteJobs", async () => {
    const onDeleteJobs = vi.fn().mockResolvedValue(undefined);
    render(<JobList {...defaultProps} onDeleteJobs={onDeleteJobs} />);

    await userEvent.click(
      screen.getByRole("checkbox", { name: /select all visible jobs/i })
    );
    await userEvent.click(screen.getByRole("button", { name: /^delete$/i }));
    await userEvent.click(screen.getByRole("button", { name: /delete 2 jobs/i }));

    expect(onDeleteJobs).toHaveBeenCalledWith(["j1", "j2"]);
  });

  it("shows only favorites when favorites-only mode is enabled", async () => {
    render(
      <JobList
        {...defaultProps}
        jobs={[fakeJob("j1", "Go Dev"), { ...fakeJob("j2", "React Dev", "f2"), IsFavorite: true }]}
      />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /show only favorite jobs/i })
    );

    expect(screen.queryByText("Go Dev")).not.toBeInTheDocument();
    expect(screen.getByText("React Dev")).toBeInTheDocument();
  });

  it("shows no-favorites empty state when favorites-only has no matches", async () => {
    render(<JobList {...defaultProps} />);

    await userEvent.click(
      screen.getByRole("button", { name: /show only favorite jobs/i })
    );

    expect(screen.getByText("No favorite jobs")).toBeInTheDocument();
    expect(
      screen.getByText("Mark jobs as favorites to quickly return to them.")
    ).toBeInTheDocument();
  });

  it("calls onToggleFavoriteJob from row star button", async () => {
    const onToggleFavoriteJob = vi.fn();
    render(
      <JobList
        {...defaultProps}
        onToggleFavoriteJob={onToggleFavoriteJob}
      />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /add go dev to favorites/i })
    );

    expect(onToggleFavoriteJob).toHaveBeenCalledWith("j1");
  });

  it("supports shift-select to apply range selection", async () => {
    const jobs = [
      fakeJob("j1", "Job 1"),
      fakeJob("j2", "Job 2"),
      fakeJob("j3", "Job 3"),
      fakeJob("j4", "Job 4"),
    ];
    render(<JobList {...defaultProps} jobs={jobs} />);

    await userEvent.click(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    );
    fireEvent.click(
      screen.getByRole("checkbox", { name: /select job job 3/i }),
      { shiftKey: true }
    );

    expect(screen.getByText("3 selected")).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 2/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 3/i })
    ).toBeChecked();
  });

  it("supports shift-select with checkbox from selected row anchor", () => {
    const jobs = [
      fakeJob("j1", "Job 1"),
      fakeJob("j2", "Job 2"),
      fakeJob("j3", "Job 3"),
      fakeJob("j4", "Job 4"),
    ];
    render(<JobList {...defaultProps} jobs={jobs} selectedJobId="j1" />);

    fireEvent.click(
      screen.getByRole("checkbox", { name: /select job job 3/i }),
      { shiftKey: true }
    );

    expect(screen.getByText("3 selected")).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 2/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 3/i })
    ).toBeChecked();
  });

  it("supports shift-select with checkbox when click event shiftKey is missing", async () => {
    const jobs = [
      fakeJob("j1", "Job 1"),
      fakeJob("j2", "Job 2"),
      fakeJob("j3", "Job 3"),
      fakeJob("j4", "Job 4"),
    ];
    render(<JobList {...defaultProps} jobs={jobs} />);

    await userEvent.click(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    );
    fireEvent.keyDown(window, { key: "Shift" });
    await userEvent.click(
      screen.getByRole("checkbox", { name: /select job job 3/i })
    );
    fireEvent.keyUp(window, { key: "Shift" });

    expect(screen.getByText("3 selected")).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 2/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 3/i })
    ).toBeChecked();
  });

  it("keeps shift-range anchor across job list refreshes", async () => {
    const initialJobs = [
      fakeJob("j1", "Job 1"),
      fakeJob("j2", "Job 2"),
      fakeJob("j3", "Job 3"),
      fakeJob("j4", "Job 4"),
      fakeJob("j5", "Job 5"),
    ];
    const { rerender } = render(<JobList {...defaultProps} jobs={initialJobs} />);

    await userEvent.click(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    );

    const refreshedJobs = [...initialJobs, fakeJob("j6", "Job 6")];
    rerender(<JobList {...defaultProps} jobs={refreshedJobs} />);

    fireEvent.click(
      screen.getByRole("checkbox", { name: /select job job 5/i }),
      { shiftKey: true }
    );

    expect(screen.getByText("5 selected")).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 2/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 3/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 4/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 5/i })
    ).toBeChecked();
  });

  it("keeps selecting ranges on repeated checkbox shift-select flows", async () => {
    const jobs = [
      fakeJob("j1", "Job 1"),
      fakeJob("j2", "Job 2"),
      fakeJob("j3", "Job 3"),
      fakeJob("j4", "Job 4"),
      fakeJob("j5", "Job 5"),
    ];
    render(<JobList {...defaultProps} jobs={jobs} />);

    await userEvent.click(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    );
    fireEvent.keyDown(window, { key: "Shift" });
    await userEvent.click(
      screen.getByRole("checkbox", { name: /select job job 5/i })
    );
    fireEvent.keyUp(window, { key: "Shift" });
    expect(screen.getByText("5 selected")).toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    );
    fireEvent.keyDown(window, { key: "Shift" });
    await userEvent.click(
      screen.getByRole("checkbox", { name: /select job job 5/i })
    );
    fireEvent.keyUp(window, { key: "Shift" });

    expect(screen.getByText("5 selected")).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 2/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 3/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 4/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 5/i })
    ).toBeChecked();
  });

  it("supports shift-select range when shift-clicking job rows", () => {
    const onSelectJob = vi.fn();
    const jobs = [
      fakeJob("j1", "Job 1"),
      fakeJob("j2", "Job 2"),
      fakeJob("j3", "Job 3"),
      fakeJob("j4", "Job 4"),
    ];
    render(
      <JobList
        {...defaultProps}
        jobs={jobs}
        selectedJobId="j1"
        onSelectJob={onSelectJob}
      />
    );

    fireEvent.click(screen.getByText("Job 3"), { shiftKey: true });

    expect(screen.getByText("3 selected")).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: /select job job 1/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 2/i })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: /select job job 3/i })
    ).toBeChecked();
    expect(onSelectJob).toHaveBeenCalledWith("j3");
  });

  it("virtualizes a 10k job list and can jump to deep rows", async () => {
    const jobs = Array.from({ length: 10_000 }, (_, index) =>
      fakeJob(`j${index + 1}`, `Job ${index + 1}`)
    );
    const { rerender, container } = render(
      <JobList {...defaultProps} jobs={jobs} />
    );

    expect(screen.getByText("10000 jobs")).toBeInTheDocument();
    // Virtualization should keep mounted rows bounded regardless of dataset size.
    expect(container.querySelectorAll('input[type="checkbox"]').length).toBeLessThan(200);

    rerender(
      <JobList {...defaultProps} jobs={jobs} selectedJobId="j10000" />
    );

    expect(await screen.findByText("Job 10000")).toBeInTheDocument();
  });
});
