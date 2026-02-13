import { useState, useEffect, useRef, useCallback } from "react";

export interface UseKeyboardShortcutsConfig {
  enabled: boolean;
  filteredJobIds: string[];
  selectedJobId: string | null;
  settingsOpen: boolean;
  jobDetailOpen: boolean;
  onSelectJob: (id: string | null) => void;
  onCloseJobDetail: () => void;
  onCloseSettings: () => void;
  onOpenSettings: () => void;
  onPollNow: () => void;
  onDeleteJob: (id: string) => Promise<void>;
  onFocusSearch: () => void;
  isPolling: boolean;
  canPoll: boolean;
}

export interface UseKeyboardShortcutsReturn {
  pendingDeleteId: string | null;
  clearPendingDelete: () => void;
  shortcutsHelpOpen: boolean;
  closeShortcutsHelp: () => void;
}

export function useKeyboardShortcuts(
  config: UseKeyboardShortcutsConfig
): UseKeyboardShortcutsReturn {
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);
  const [shortcutsHelpOpen, setShortcutsHelpOpen] = useState(false);
  const configRef = useRef(config);
  const deleteTimerRef = useRef<number | null>(null);
  const pendingDeleteRef = useRef(pendingDeleteId);
  const shortcutsHelpRef = useRef(shortcutsHelpOpen);

  // Keep refs in sync with latest values.
  useEffect(() => {
    configRef.current = config;
  });
  useEffect(() => {
    pendingDeleteRef.current = pendingDeleteId;
  });
  useEffect(() => {
    shortcutsHelpRef.current = shortcutsHelpOpen;
  });

  // Auto-clear pending delete when selection changes away from pending job.
  useEffect(() => {
    if (
      pendingDeleteId !== null &&
      config.selectedJobId !== pendingDeleteId
    ) {
      setPendingDeleteId(null);
      if (deleteTimerRef.current !== null) {
        clearTimeout(deleteTimerRef.current);
        deleteTimerRef.current = null;
      }
    }
  }, [config.selectedJobId, pendingDeleteId]);

  const clearPendingDelete = useCallback(() => {
    setPendingDeleteId(null);
    if (deleteTimerRef.current !== null) {
      clearTimeout(deleteTimerRef.current);
      deleteTimerRef.current = null;
    }
  }, []);

  const closeShortcutsHelp = useCallback(() => {
    setShortcutsHelpOpen(false);
  }, []);

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const c = configRef.current;
      const target = e.target as HTMLElement;
      const tag = target.tagName;
      const isInput =
        tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
      const hasModifier = e.metaKey || e.ctrlKey;

      // Escape always works, even when input focused or shortcuts disabled.
      if (e.key === "Escape") {
        if (isInput) {
          target.blur();
          e.preventDefault();
          return;
        }
        if (shortcutsHelpRef.current) {
          setShortcutsHelpOpen(false);
          e.preventDefault();
          return;
        }
        if (pendingDeleteRef.current !== null) {
          setPendingDeleteId(null);
          if (deleteTimerRef.current !== null) {
            clearTimeout(deleteTimerRef.current);
            deleteTimerRef.current = null;
          }
          e.preventDefault();
          return;
        }
        if (c.settingsOpen) {
          c.onCloseSettings();
          e.preventDefault();
          return;
        }
        if (c.jobDetailOpen) {
          c.onCloseJobDetail();
          e.preventDefault();
          return;
        }
        return;
      }

      // All other shortcuts require enabled and no input focus.
      if (!c.enabled) return;

      // Ctrl+F / Cmd+F and Ctrl+R / Cmd+R work even when input is focused.
      if (hasModifier && e.key === "f") {
        c.onFocusSearch();
        e.preventDefault();
        return;
      }
      if (hasModifier && e.key === "r") {
        if (c.canPoll && !c.isPolling) {
          c.onPollNow();
        }
        e.preventDefault();
        return;
      }

      // Skip all other shortcuts when input is focused.
      if (isInput) return;

      switch (e.key) {
        case "j":
        case "ArrowDown":
          navigate(c, 1);
          e.preventDefault();
          break;
        case "k":
        case "ArrowUp":
          navigate(c, -1);
          e.preventDefault();
          break;
        case "Enter":
          if (c.selectedJobId) {
            c.onSelectJob(c.selectedJobId);
            e.preventDefault();
          }
          break;
        case "/":
          c.onFocusSearch();
          e.preventDefault();
          break;
        case ",":
          c.onOpenSettings();
          e.preventDefault();
          break;
        case "Delete":
        case "Backspace":
          handleDelete(c);
          e.preventDefault();
          break;
        case "?":
          setShortcutsHelpOpen(true);
          e.preventDefault();
          break;
      }
    }

    function navigate(c: UseKeyboardShortcutsConfig, direction: 1 | -1) {
      const ids = c.filteredJobIds;
      if (ids.length === 0) return;

      if (c.selectedJobId === null) {
        c.onSelectJob(direction === 1 ? ids[0] : ids[ids.length - 1]);
        return;
      }

      const currentIndex = ids.indexOf(c.selectedJobId);
      if (currentIndex === -1) {
        c.onSelectJob(ids[0]);
        return;
      }

      const nextIndex = currentIndex + direction;
      if (nextIndex >= 0 && nextIndex < ids.length) {
        c.onSelectJob(ids[nextIndex]);
      }
    }

    function handleDelete(c: UseKeyboardShortcutsConfig) {
      if (!c.selectedJobId) return;

      if (pendingDeleteRef.current === c.selectedJobId) {
        // Second press — execute.
        c.onDeleteJob(c.selectedJobId);
        setPendingDeleteId(null);
        if (deleteTimerRef.current !== null) {
          clearTimeout(deleteTimerRef.current);
          deleteTimerRef.current = null;
        }
      } else {
        // First press — arm confirmation.
        setPendingDeleteId(c.selectedJobId);
        if (deleteTimerRef.current !== null) {
          clearTimeout(deleteTimerRef.current);
        }
        deleteTimerRef.current = window.setTimeout(() => {
          setPendingDeleteId(null);
          deleteTimerRef.current = null;
        }, 3000);
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, []); // Register once, read latest from refs.

  // Cleanup timer on unmount.
  useEffect(() => {
    return () => {
      if (deleteTimerRef.current !== null) {
        clearTimeout(deleteTimerRef.current);
      }
    };
  }, []);

  return { pendingDeleteId, clearPendingDelete, shortcutsHelpOpen, closeShortcutsHelp };
}
