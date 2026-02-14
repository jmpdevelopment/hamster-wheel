import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useJobs } from "./useJobs";

// Mock Wails bindings.
const mockGetJobs = vi.fn();
const mockGetJobCount = vi.fn();
const mockDeleteJob = vi.fn();
const mockSetJobFavorite = vi.fn();
const mockSetJobsFavorite = vi.fn();

vi.mock("../../bindings/hamster-wheel/jobservice", () => ({
  GetJobs: (...args: unknown[]) => mockGetJobs(...args),
  GetJobCount: (...args: unknown[]) => mockGetJobCount(...args),
  DeleteJob: (...args: unknown[]) => mockDeleteJob(...args),
  SetJobFavorite: (...args: unknown[]) => mockSetJobFavorite(...args),
  SetJobsFavorite: (...args: unknown[]) => mockSetJobsFavorite(...args),
}));

const fakeJob = (id: string, title: string) => ({
  ID: id,
  Source: "reed_uk",
  SourceID: `src-${id}`,
  Title: title,
  Company: "Acme",
  Location: "London",
  Description: "A job",
  URL: "https://example.com",
  PostedAt: "2026-02-08T10:00:00Z",
  DiscoveredAt: "2026-02-08T11:00:00Z",
  FilterID: "filter-1",
  IsFavorite: false,
});

beforeEach(() => {
  vi.clearAllMocks();
  mockGetJobs.mockResolvedValue([fakeJob("1", "Go Dev"), fakeJob("2", "React Dev")]);
  mockGetJobCount.mockResolvedValue(2);
  mockDeleteJob.mockResolvedValue(undefined);
  mockSetJobFavorite.mockResolvedValue(undefined);
  mockSetJobsFavorite.mockResolvedValue(undefined);
});

describe("useJobs", () => {
  it("fetches jobs on mount", async () => {
    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.jobs).toHaveLength(2);
    expect(result.current.jobs[0].Title).toBe("Go Dev");
    expect(result.current.jobCount).toBe(2);
    expect(result.current.error).toBeNull();
  });

  it("passes limit to GetJobs", async () => {
    renderHook(() => useJobs(25));

    await waitFor(() => expect(mockGetJobs).toHaveBeenCalledWith(25));
  });

  it("handles null response from Go (nil slice)", async () => {
    mockGetJobs.mockResolvedValue(null);
    mockGetJobCount.mockResolvedValue(null);

    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.jobs).toEqual([]);
    expect(result.current.jobCount).toBe(0);
  });

  it("sets error on fetch failure", async () => {
    mockGetJobs.mockRejectedValue(new Error("network error"));

    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe("network error");
    expect(result.current.jobs).toEqual([]);
  });

  it("refresh re-fetches data", async () => {
    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(mockGetJobs).toHaveBeenCalledTimes(1);

    await act(async () => {
      await result.current.refresh();
    });

    expect(mockGetJobs).toHaveBeenCalledTimes(2);
  });

  it("deleteJob calls DeleteJob and refreshes", async () => {
    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.deleteJob("1");
    });

    expect(mockDeleteJob).toHaveBeenCalledWith("1");
    // refresh is called after delete
    expect(mockGetJobs).toHaveBeenCalledTimes(2);
  });

  it("deleteJobs deletes all IDs and refreshes once", async () => {
    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.deleteJobs(["1", "2", "1"]);
    });

    expect(mockDeleteJob).toHaveBeenCalledTimes(2);
    expect(mockDeleteJob).toHaveBeenNthCalledWith(1, "1");
    expect(mockDeleteJob).toHaveBeenNthCalledWith(2, "2");
    expect(mockGetJobs).toHaveBeenCalledTimes(2);
  });

  it("deleteJobs reports partial failures and still refreshes", async () => {
    mockDeleteJob
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(new Error("missing"));

    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await expect(result.current.deleteJobs(["1", "2"])).rejects.toThrow(
        "Failed to delete 1 of 2 jobs: missing"
      );
    });

    expect(mockGetJobs).toHaveBeenCalledTimes(2);
    expect(result.current.error).toBe("Failed to delete 1 of 2 jobs: missing");
  });

  it("deleteJobs retries transient SQLITE_BUSY errors", async () => {
    mockDeleteJob
      .mockRejectedValueOnce(new Error("database is locked (5) (SQLITE_BUSY)"))
      .mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.deleteJobs(["1"]);
    });

    expect(mockDeleteJob).toHaveBeenCalledTimes(2);
    expect(mockGetJobs).toHaveBeenCalledTimes(2);
    expect(result.current.error).toBeNull();
  });

  it("setJobFavorite updates favorite and refreshes", async () => {
    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.setJobFavorite("1", true);
    });

    expect(mockSetJobFavorite).toHaveBeenCalledWith("1", true);
    expect(mockGetJobs).toHaveBeenCalledTimes(2);
  });

  it("setJobsFavorite updates favorites in bulk and refreshes", async () => {
    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.setJobsFavorite(["1", "2", "1"], false);
    });

    expect(mockSetJobsFavorite).toHaveBeenCalledWith(["1", "2"], false);
    expect(mockGetJobs).toHaveBeenCalledTimes(2);
  });

  it("deleteJob sets error on failure", async () => {
    mockDeleteJob.mockRejectedValue(new Error("not found"));

    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await expect(result.current.deleteJob("999")).rejects.toThrow(
        "not found"
      );
    });

    expect(result.current.error).toBe("not found");
  });

  it("clears previous error on successful refresh", async () => {
    mockGetJobs.mockRejectedValueOnce(new Error("fail"));

    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.error).toBe("fail"));

    mockGetJobs.mockResolvedValue([fakeJob("1", "Go Dev")]);

    await act(async () => {
      await result.current.refresh();
    });

    expect(result.current.error).toBeNull();
  });

  it("clearError clears the current error", async () => {
    mockGetJobs.mockRejectedValue(new Error("network error"));

    const { result } = renderHook(() => useJobs());

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe("network error");

    act(() => {
      result.current.clearError();
    });

    expect(result.current.error).toBeNull();
  });
});
