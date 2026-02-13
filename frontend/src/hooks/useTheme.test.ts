import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useTheme } from "./useTheme";

// Mock Wails bindings.
const mockGetTheme = vi.fn();
const mockSetTheme = vi.fn();

vi.mock("../../bindings/hamster-wheel/settingsservice", () => ({
  GetTheme: (...args: unknown[]) => mockGetTheme(...args),
  SetTheme: (...args: unknown[]) => mockSetTheme(...args),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mockGetTheme.mockResolvedValue("");
  mockSetTheme.mockResolvedValue(undefined);
  // Reset <html> class.
  document.documentElement.className = "dark";
});

describe("useTheme", () => {
  it("defaults to system when no theme is saved", async () => {
    const { result } = renderHook(() => useTheme());
    await waitFor(() => {
      expect(result.current.theme).toBe("system");
    });
  });

  it("loads saved dark theme on mount", async () => {
    mockGetTheme.mockResolvedValue("dark");
    const { result } = renderHook(() => useTheme());
    await waitFor(() => {
      expect(result.current.theme).toBe("dark");
    });
  });

  it("loads saved light theme on mount", async () => {
    mockGetTheme.mockResolvedValue("light");
    const { result } = renderHook(() => useTheme());
    await waitFor(() => {
      expect(result.current.theme).toBe("light");
    });
  });

  it("applies light class when theme is light", async () => {
    mockGetTheme.mockResolvedValue("light");
    renderHook(() => useTheme());
    await waitFor(() => {
      expect(document.documentElement.classList.contains("light")).toBe(true);
      expect(document.documentElement.classList.contains("dark")).toBe(false);
    });
  });

  it("applies dark class when theme is dark", async () => {
    mockGetTheme.mockResolvedValue("dark");
    renderHook(() => useTheme());
    await waitFor(() => {
      expect(document.documentElement.classList.contains("dark")).toBe(true);
      expect(document.documentElement.classList.contains("light")).toBe(false);
    });
  });

  it("treats unknown saved value as system", async () => {
    mockGetTheme.mockResolvedValue("invalid");
    const { result } = renderHook(() => useTheme());
    await waitFor(() => {
      expect(result.current.theme).toBe("system");
    });
  });

  it("persists theme via SetTheme when setTheme is called", async () => {
    const { result } = renderHook(() => useTheme());
    await waitFor(() => {
      expect(result.current.theme).toBe("system");
    });

    await act(async () => {
      await result.current.setTheme("light");
    });

    expect(mockSetTheme).toHaveBeenCalledWith("light");
    expect(result.current.theme).toBe("light");
  });

  it("applies classes immediately on setTheme", async () => {
    const { result } = renderHook(() => useTheme());
    await waitFor(() => {
      expect(result.current.theme).toBe("system");
    });

    await act(async () => {
      await result.current.setTheme("light");
    });

    expect(document.documentElement.classList.contains("light")).toBe(true);
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("handles save errors gracefully", async () => {
    mockSetTheme.mockRejectedValue(new Error("db error"));
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    const { result } = renderHook(() => useTheme());
    await waitFor(() => {
      expect(result.current.theme).toBe("system");
    });

    await act(async () => {
      await result.current.setTheme("dark");
    });

    // Theme should still be applied even if save fails.
    expect(result.current.theme).toBe("dark");
    consoleSpy.mockRestore();
  });

  it("handles load errors gracefully", async () => {
    mockGetTheme.mockRejectedValue(new Error("load error"));
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    const { result } = renderHook(() => useTheme());
    await waitFor(() => {
      expect(result.current.theme).toBe("system");
    });

    consoleSpy.mockRestore();
  });
});
