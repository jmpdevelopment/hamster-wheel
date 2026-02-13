import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { useJobSearch } from "./useJobSearch";

const fakeJob = (
  id: string,
  title: string,
  opts: Partial<{
    company: string;
    location: string;
    description: string;
    filterId: string;
  }> = {}
) => ({
  ID: id,
  Source: "reed_uk",
  SourceID: `src-${id}`,
  Title: title,
  Company: opts.company ?? "Acme",
  Location: opts.location ?? "London",
  Description: opts.description ?? "A job",
  URL: "https://example.com",
  PostedAt: "2026-02-08T10:00:00Z",
  DiscoveredAt: "2026-02-08T11:00:00Z",
  FilterID: opts.filterId ?? "f1",
});

describe("useJobSearch", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns all jobs when search is empty and no filter", () => {
    const jobs = [fakeJob("1", "Go Dev"), fakeJob("2", "React Dev")];
    const { result } = renderHook(() => useJobSearch(jobs, null));
    expect(result.current.filteredJobs).toHaveLength(2);
  });

  it("filters by filterByFilterId", () => {
    const jobs = [
      fakeJob("1", "Go Dev", { filterId: "f1" }),
      fakeJob("2", "React Dev", { filterId: "f2" }),
    ];
    const { result } = renderHook(() => useJobSearch(jobs, "f1"));
    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].Title).toBe("Go Dev");
  });

  it("single-term search matches title (case-insensitive)", () => {
    const jobs = [fakeJob("1", "Go Developer"), fakeJob("2", "React Dev")];
    const { result } = renderHook(() => useJobSearch(jobs, null));

    act(() => {
      result.current.setSearchTerm("go");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].Title).toBe("Go Developer");
  });

  it("multi-term search requires all terms to match", () => {
    const jobs = [
      fakeJob("1", "Senior Go Developer"),
      fakeJob("2", "Junior Go Engineer"),
      fakeJob("3", "React Developer"),
    ];
    const { result } = renderHook(() => useJobSearch(jobs, null));

    act(() => {
      result.current.setSearchTerm("go dev");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].Title).toBe("Senior Go Developer");
  });

  it("matches against company field", () => {
    const jobs = [
      fakeJob("1", "Dev", { company: "Google" }),
      fakeJob("2", "Dev", { company: "Meta" }),
    ];
    const { result } = renderHook(() => useJobSearch(jobs, null));

    act(() => {
      result.current.setSearchTerm("google");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].Company).toBe("Google");
  });

  it("matches against location field", () => {
    const jobs = [
      fakeJob("1", "Dev", { location: "Manchester" }),
      fakeJob("2", "Dev", { location: "London" }),
    ];
    const { result } = renderHook(() => useJobSearch(jobs, null));

    act(() => {
      result.current.setSearchTerm("manchester");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].Location).toBe("Manchester");
  });

  it("matches against description field", () => {
    const jobs = [
      fakeJob("1", "Dev", { description: "Work with Kubernetes clusters" }),
      fakeJob("2", "Dev", { description: "Build web apps" }),
    ];
    const { result } = renderHook(() => useJobSearch(jobs, null));

    act(() => {
      result.current.setSearchTerm("kubernetes");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].Description).toBe(
      "Work with Kubernetes clusters"
    );
  });

  it("is case-insensitive", () => {
    const jobs = [fakeJob("1", "Senior GO Developer")];
    const { result } = renderHook(() => useJobSearch(jobs, null));

    act(() => {
      result.current.setSearchTerm("GO DEVELOPER");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current.filteredJobs).toHaveLength(1);
  });

  it("whitespace-only search returns all jobs", () => {
    const jobs = [fakeJob("1", "Go Dev"), fakeJob("2", "React Dev")];
    const { result } = renderHook(() => useJobSearch(jobs, null));

    act(() => {
      result.current.setSearchTerm("   ");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current.filteredJobs).toHaveLength(2);
  });

  it("debounces: does not filter immediately", () => {
    const jobs = [fakeJob("1", "Go Dev"), fakeJob("2", "React Dev")];
    const { result } = renderHook(() => useJobSearch(jobs, null));

    act(() => {
      result.current.setSearchTerm("react");
    });

    // Before debounce fires
    expect(result.current.filteredJobs).toHaveLength(2);
  });

  it("debounces: filters after 200ms", () => {
    const jobs = [fakeJob("1", "Go Dev"), fakeJob("2", "React Dev")];
    const { result } = renderHook(() => useJobSearch(jobs, null));

    act(() => {
      result.current.setSearchTerm("react");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].Title).toBe("React Dev");
  });

  it("combines search and filter", () => {
    const jobs = [
      fakeJob("1", "Go Dev", { filterId: "f1" }),
      fakeJob("2", "React Dev", { filterId: "f2" }),
      fakeJob("3", "Go Engineer", { filterId: "f2" }),
    ];
    const { result } = renderHook(() => useJobSearch(jobs, "f2"));

    act(() => {
      result.current.setSearchTerm("go");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].Title).toBe("Go Engineer");
  });

  it("clearing search restores filter-only results", () => {
    const jobs = [
      fakeJob("1", "Go Dev", { filterId: "f1" }),
      fakeJob("2", "React Dev", { filterId: "f1" }),
    ];
    const { result } = renderHook(() => useJobSearch(jobs, "f1"));

    act(() => {
      result.current.setSearchTerm("go");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(result.current.filteredJobs).toHaveLength(1);

    act(() => {
      result.current.setSearchTerm("");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(result.current.filteredJobs).toHaveLength(2);
  });

  it("updates results when jobs array changes", () => {
    const initialJobs = [fakeJob("1", "Go Dev")];
    const { result, rerender } = renderHook(
      ({ jobs }) => useJobSearch(jobs, null),
      { initialProps: { jobs: initialJobs } }
    );

    act(() => {
      result.current.setSearchTerm("react");
    });
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(result.current.filteredJobs).toHaveLength(0);

    const updatedJobs = [...initialJobs, fakeJob("2", "React Dev")];
    rerender({ jobs: updatedJobs });

    expect(result.current.filteredJobs).toHaveLength(1);
    expect(result.current.filteredJobs[0].Title).toBe("React Dev");
  });
});
