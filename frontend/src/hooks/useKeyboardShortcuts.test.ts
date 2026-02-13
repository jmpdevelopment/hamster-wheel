import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  useKeyboardShortcuts,
  UseKeyboardShortcutsConfig,
} from "./useKeyboardShortcuts";

function makeConfig(
  overrides: Partial<UseKeyboardShortcutsConfig> = {}
): UseKeyboardShortcutsConfig {
  return {
    enabled: true,
    filteredJobIds: ["j1", "j2", "j3"],
    selectedJobId: null,
    settingsOpen: false,
    jobDetailOpen: false,
    onSelectJob: vi.fn(),
    onCloseJobDetail: vi.fn(),
    onCloseSettings: vi.fn(),
    onOpenSettings: vi.fn(),
    onPollNow: vi.fn(),
    onDeleteJob: vi.fn().mockResolvedValue(undefined),
    onFocusSearch: vi.fn(),
    isPolling: false,
    canPoll: true,
    ...overrides,
  };
}

function pressKey(
  key: string,
  opts: Partial<KeyboardEventInit> = {}
) {
  const event = new KeyboardEvent("keydown", {
    key,
    bubbles: true,
    ...opts,
  });
  document.dispatchEvent(event);
  return event;
}

describe("useKeyboardShortcuts", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  // --- Enabled/Disabled ---

  it("ignores shortcuts (except Escape) when disabled", () => {
    const config = makeConfig({ enabled: false, selectedJobId: "j1" });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("j"));
    act(() => pressKey("/"));
    act(() => pressKey(","));

    expect(config.onSelectJob).not.toHaveBeenCalled();
    expect(config.onFocusSearch).not.toHaveBeenCalled();
    expect(config.onOpenSettings).not.toHaveBeenCalled();
  });

  it("Escape still works when disabled", () => {
    const config = makeConfig({ enabled: false, settingsOpen: true });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Escape"));

    expect(config.onCloseSettings).toHaveBeenCalled();
  });

  // --- Input Gating ---

  it("ignores shortcuts when an input is focused", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => {
      input.dispatchEvent(
        new KeyboardEvent("keydown", { key: "j", bubbles: true })
      );
    });

    expect(config.onSelectJob).not.toHaveBeenCalled();
    document.body.removeChild(input);
  });

  it("ignores shortcuts when a textarea is focused", () => {
    const textarea = document.createElement("textarea");
    document.body.appendChild(textarea);
    textarea.focus();

    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => {
      textarea.dispatchEvent(
        new KeyboardEvent("keydown", { key: "j", bubbles: true })
      );
    });

    expect(config.onSelectJob).not.toHaveBeenCalled();
    document.body.removeChild(textarea);
  });

  it("ignores shortcuts when a select is focused", () => {
    const select = document.createElement("select");
    document.body.appendChild(select);
    select.focus();

    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => {
      select.dispatchEvent(
        new KeyboardEvent("keydown", { key: "j", bubbles: true })
      );
    });

    expect(config.onSelectJob).not.toHaveBeenCalled();
    document.body.removeChild(select);
  });

  // --- Escape ---

  it("Escape blurs focused input", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => {
      input.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape", bubbles: true })
      );
    });

    expect(document.activeElement).not.toBe(input);
    document.body.removeChild(input);
  });

  it("Escape closes settings panel", () => {
    const config = makeConfig({ settingsOpen: true });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Escape"));

    expect(config.onCloseSettings).toHaveBeenCalled();
    expect(config.onCloseJobDetail).not.toHaveBeenCalled();
  });

  it("Escape closes job detail when settings is closed", () => {
    const config = makeConfig({ jobDetailOpen: true });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Escape"));

    expect(config.onCloseJobDetail).toHaveBeenCalled();
  });

  it("Escape cancels pending delete before closing panels", () => {
    const config = makeConfig({ selectedJobId: "j1", settingsOpen: true });
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    // Arm delete.
    act(() => pressKey("Delete"));
    expect(result.current.pendingDeleteId).toBe("j1");

    // Escape should cancel pending delete, not close settings.
    act(() => pressKey("Escape"));
    expect(result.current.pendingDeleteId).toBeNull();
    expect(config.onCloseSettings).not.toHaveBeenCalled();
  });

  it("Escape does nothing when nothing is open", () => {
    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Escape"));

    expect(config.onCloseSettings).not.toHaveBeenCalled();
    expect(config.onCloseJobDetail).not.toHaveBeenCalled();
  });

  // --- j/k Navigation ---

  it("j selects first job when nothing is selected", () => {
    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("j"));

    expect(config.onSelectJob).toHaveBeenCalledWith("j1");
  });

  it("k selects last job when nothing is selected", () => {
    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("k"));

    expect(config.onSelectJob).toHaveBeenCalledWith("j3");
  });

  it("j moves selection forward by one", () => {
    const config = makeConfig({ selectedJobId: "j1" });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("j"));

    expect(config.onSelectJob).toHaveBeenCalledWith("j2");
  });

  it("k moves selection backward by one", () => {
    const config = makeConfig({ selectedJobId: "j2" });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("k"));

    expect(config.onSelectJob).toHaveBeenCalledWith("j1");
  });

  it("ArrowDown works same as j", () => {
    const config = makeConfig({ selectedJobId: "j1" });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("ArrowDown"));

    expect(config.onSelectJob).toHaveBeenCalledWith("j2");
  });

  it("ArrowUp works same as k", () => {
    const config = makeConfig({ selectedJobId: "j2" });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("ArrowUp"));

    expect(config.onSelectJob).toHaveBeenCalledWith("j1");
  });

  it("j at end of list does not wrap", () => {
    const config = makeConfig({ selectedJobId: "j3" });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("j"));

    expect(config.onSelectJob).not.toHaveBeenCalled();
  });

  it("k at start of list does not wrap", () => {
    const config = makeConfig({ selectedJobId: "j1" });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("k"));

    expect(config.onSelectJob).not.toHaveBeenCalled();
  });

  it("navigation with empty list does nothing", () => {
    const config = makeConfig({ filteredJobIds: [] });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("j"));
    act(() => pressKey("k"));

    expect(config.onSelectJob).not.toHaveBeenCalled();
  });

  it("j selects first when selected job is not in filtered list", () => {
    const config = makeConfig({ selectedJobId: "j99" });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("j"));

    expect(config.onSelectJob).toHaveBeenCalledWith("j1");
  });

  // --- Enter ---

  it("Enter opens selected job detail", () => {
    const config = makeConfig({ selectedJobId: "j2" });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Enter"));

    expect(config.onSelectJob).toHaveBeenCalledWith("j2");
  });

  it("Enter does nothing when no job is selected", () => {
    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Enter"));

    expect(config.onSelectJob).not.toHaveBeenCalled();
  });

  // --- Ctrl+R / Cmd+R ---

  it("Ctrl+R calls onPollNow when canPoll and not polling", () => {
    const config = makeConfig({ canPoll: true, isPolling: false });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("r", { ctrlKey: true }));

    expect(config.onPollNow).toHaveBeenCalled();
  });

  it("Cmd+R calls onPollNow (macOS)", () => {
    const config = makeConfig({ canPoll: true, isPolling: false });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("r", { metaKey: true }));

    expect(config.onPollNow).toHaveBeenCalled();
  });

  it("Ctrl+R does nothing when canPoll is false", () => {
    const config = makeConfig({ canPoll: false });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("r", { ctrlKey: true }));

    expect(config.onPollNow).not.toHaveBeenCalled();
  });

  it("Ctrl+R does nothing when already polling", () => {
    const config = makeConfig({ isPolling: true });
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("r", { ctrlKey: true }));

    expect(config.onPollNow).not.toHaveBeenCalled();
  });

  // --- / and Ctrl+F / Cmd+F ---

  it("/ focuses search bar", () => {
    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("/"));

    expect(config.onFocusSearch).toHaveBeenCalled();
  });

  it("Ctrl+F focuses search bar", () => {
    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("f", { ctrlKey: true }));

    expect(config.onFocusSearch).toHaveBeenCalled();
  });

  it("Cmd+F focuses search bar (macOS)", () => {
    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("f", { metaKey: true }));

    expect(config.onFocusSearch).toHaveBeenCalled();
  });

  it("Ctrl+F works even when input is focused", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => {
      input.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "f",
          ctrlKey: true,
          bubbles: true,
        })
      );
    });

    expect(config.onFocusSearch).toHaveBeenCalled();
    document.body.removeChild(input);
  });

  // --- , (settings) ---

  it(", opens settings", () => {
    const config = makeConfig();
    renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey(","));

    expect(config.onOpenSettings).toHaveBeenCalled();
  });

  // --- Delete / Backspace ---

  it("first Delete sets pendingDeleteId", () => {
    const config = makeConfig({ selectedJobId: "j1" });
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Delete"));

    expect(result.current.pendingDeleteId).toBe("j1");
    expect(config.onDeleteJob).not.toHaveBeenCalled();
  });

  it("second Delete within 3s calls onDeleteJob", () => {
    const config = makeConfig({ selectedJobId: "j1" });
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Delete"));
    expect(result.current.pendingDeleteId).toBe("j1");

    act(() => pressKey("Delete"));
    expect(config.onDeleteJob).toHaveBeenCalledWith("j1");
    expect(result.current.pendingDeleteId).toBeNull();
  });

  it("pendingDeleteId auto-clears after 3s", () => {
    const config = makeConfig({ selectedJobId: "j1" });
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Delete"));
    expect(result.current.pendingDeleteId).toBe("j1");

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(result.current.pendingDeleteId).toBeNull();
  });

  it("Backspace works the same as Delete", () => {
    const config = makeConfig({ selectedJobId: "j1" });
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Backspace"));
    expect(result.current.pendingDeleteId).toBe("j1");

    act(() => pressKey("Backspace"));
    expect(config.onDeleteJob).toHaveBeenCalledWith("j1");
    expect(result.current.pendingDeleteId).toBeNull();
  });

  it("Delete does nothing when no job is selected", () => {
    const config = makeConfig();
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Delete"));

    expect(result.current.pendingDeleteId).toBeNull();
    expect(config.onDeleteJob).not.toHaveBeenCalled();
  });

  // --- ? (Shortcuts Help) ---

  it("? opens shortcuts help", () => {
    const config = makeConfig();
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    expect(result.current.shortcutsHelpOpen).toBe(false);

    act(() => pressKey("?"));

    expect(result.current.shortcutsHelpOpen).toBe(true);
  });

  it("Escape closes shortcuts help", () => {
    const config = makeConfig();
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("?"));
    expect(result.current.shortcutsHelpOpen).toBe(true);

    act(() => pressKey("Escape"));
    expect(result.current.shortcutsHelpOpen).toBe(false);
  });

  it("Escape closes shortcuts help before settings", () => {
    const config = makeConfig({ settingsOpen: true });
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("?"));
    expect(result.current.shortcutsHelpOpen).toBe(true);

    act(() => pressKey("Escape"));
    expect(result.current.shortcutsHelpOpen).toBe(false);
    expect(config.onCloseSettings).not.toHaveBeenCalled();
  });

  it("closeShortcutsHelp closes shortcuts help", () => {
    const config = makeConfig();
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("?"));
    expect(result.current.shortcutsHelpOpen).toBe(true);

    act(() => result.current.closeShortcutsHelp());
    expect(result.current.shortcutsHelpOpen).toBe(false);
  });

  it("? is ignored when disabled", () => {
    const config = makeConfig({ enabled: false });
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("?"));

    expect(result.current.shortcutsHelpOpen).toBe(false);
  });

  it("clearPendingDelete clears the pending state", () => {
    const config = makeConfig({ selectedJobId: "j1" });
    const { result } = renderHook(() => useKeyboardShortcuts(config));

    act(() => pressKey("Delete"));
    expect(result.current.pendingDeleteId).toBe("j1");

    act(() => result.current.clearPendingDelete());
    expect(result.current.pendingDeleteId).toBeNull();
  });
});
