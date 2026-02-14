import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useFilters } from "./useFilters";

// Mock Wails bindings.
const mockGetFilters = vi.fn();
const mockCreateFilter = vi.fn();
const mockUpdateFilter = vi.fn();
const mockDeleteFilter = vi.fn();

vi.mock("../../bindings/hamster-wheel/filterservice", () => ({
  GetFilters: (...args: unknown[]) => mockGetFilters(...args),
  CreateFilter: (...args: unknown[]) => mockCreateFilter(...args),
  UpdateFilter: (...args: unknown[]) => mockUpdateFilter(...args),
  DeleteFilter: (...args: unknown[]) => mockDeleteFilter(...args),
}));

const fakeFilter = (id: string, name: string, enabled = true) => ({
  ID: id,
  Name: name,
  Keywords: "go developer",
  Location: "London",
  Source: "reed_uk",
  Enabled: enabled,
  CreatedAt: "2026-02-08T10:00:00Z",
  UpdatedAt: "2026-02-08T10:00:00Z",
});

beforeEach(() => {
  vi.clearAllMocks();
  mockGetFilters.mockResolvedValue([
    fakeFilter("f1", "Backend London"),
    fakeFilter("f2", "Remote React", false),
  ]);
  mockCreateFilter.mockResolvedValue("f3");
  mockUpdateFilter.mockResolvedValue(undefined);
  mockDeleteFilter.mockResolvedValue(undefined);
});

describe("useFilters", () => {
  it("fetches filters on mount", async () => {
    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.filters).toHaveLength(2);
    expect(result.current.filters[0].Name).toBe("Backend London");
    expect(result.current.error).toBeNull();
  });

  it("handles null response from Go (nil slice)", async () => {
    mockGetFilters.mockResolvedValue(null);

    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.filters).toEqual([]);
  });

  it("sets error on fetch failure", async () => {
    mockGetFilters.mockRejectedValue(new Error("db closed"));

    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe("db closed");
    expect(result.current.filters).toEqual([]);
  });

  it("createFilter calls CreateFilter and refreshes", async () => {
    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));

    let id: string = "";
    await act(async () => {
      id = await result.current.createFilter(
        "New Filter",
        "typescript",
        "Remote",
        "reed_uk"
      );
    });

    expect(id).toBe("f3");
    expect(mockCreateFilter).toHaveBeenCalledWith(
      "New Filter",
      "typescript",
      "Remote",
      "reed_uk"
    );
    // refresh after create
    expect(mockGetFilters).toHaveBeenCalledTimes(2);
  });

  it("createFilter sets error on failure", async () => {
    mockCreateFilter.mockRejectedValue(new Error("duplicate"));

    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      try {
        await result.current.createFilter("Dup", "kw", "loc", "src");
      } catch {
        // Expected — createFilter re-throws so callers can handle it.
      }
    });

    expect(result.current.error).toBe("duplicate");
  });

  it("updateFilter calls UpdateFilter and refreshes", async () => {
    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.updateFilter(
        "f1",
        "Updated Name",
        "react",
        "Manchester",
        "reed_uk",
        false
      );
    });

    expect(mockUpdateFilter).toHaveBeenCalledWith(
      "f1",
      "Updated Name",
      "react",
      "Manchester",
      "reed_uk",
      false
    );
    expect(mockGetFilters).toHaveBeenCalledTimes(2);
  });

  it("updateFilter sets error on failure", async () => {
    mockUpdateFilter.mockRejectedValue(new Error("not found"));

    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await expect(
        result.current.updateFilter(
          "f999",
          "n",
          "k",
          "l",
          "s",
          true
        )
      ).rejects.toThrow("not found");
    });

    expect(result.current.error).toBe("not found");
  });

  it("deleteFilter calls DeleteFilter and refreshes", async () => {
    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.deleteFilter("f1");
    });

    expect(mockDeleteFilter).toHaveBeenCalledWith("f1");
    expect(mockGetFilters).toHaveBeenCalledTimes(2);
  });

  it("deleteFilter sets error on failure", async () => {
    mockDeleteFilter.mockRejectedValue(new Error("not found"));

    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await expect(result.current.deleteFilter("f999")).rejects.toThrow(
        "not found"
      );
    });

    expect(result.current.error).toBe("not found");
  });

  it("clears previous error on successful refresh", async () => {
    mockGetFilters.mockRejectedValueOnce(new Error("fail"));

    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.error).toBe("fail"));

    mockGetFilters.mockResolvedValue([fakeFilter("f1", "Test")]);

    await act(async () => {
      await result.current.refresh();
    });

    expect(result.current.error).toBeNull();
  });

  it("clearError clears the current error", async () => {
    mockGetFilters.mockRejectedValue(new Error("db closed"));

    const { result } = renderHook(() => useFilters());

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe("db closed");

    act(() => {
      result.current.clearError();
    });

    expect(result.current.error).toBeNull();
  });
});
