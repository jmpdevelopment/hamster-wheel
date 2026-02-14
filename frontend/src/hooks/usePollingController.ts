import {
  useCallback,
  useEffect,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";
import { Events } from "@wailsio/runtime";
import {
  GetPollingStatus,
  PollNow,
  SetPollingPaused,
} from "../../bindings/hamster-wheel/pollingservice";
import { PollRunResult } from "../../bindings/hamster-wheel/models";

const schedulerSyncIntervalMs = 60_000;
const defaultPollIntervalMs = 30 * 60_000;
const pollingStatusEvent = "polling:status-changed";

type PollingStatusState = {
  paused: boolean;
  nextPollAt: string;
};

export interface UsePollingControllerOptions {
  refreshJobs: () => Promise<void>;
  refreshFilters: () => Promise<void>;
  setAppError: Dispatch<SetStateAction<string | null>>;
}

export interface UsePollingControllerReturn {
  isPolling: boolean;
  pollingPaused: boolean;
  nextPollAt: string;
  pollRun: PollRunResult | null;
  pollNow: () => Promise<void>;
  togglePolling: () => Promise<void>;
  clearPollRun: () => void;
}

const pickBoolean = (...candidates: unknown[]): boolean | undefined => {
  return candidates.find(
    (candidate): candidate is boolean => typeof candidate === "boolean"
  );
};

const pickNonEmptyString = (...candidates: unknown[]): string | undefined => {
  return candidates.find(
    (candidate): candidate is string =>
      typeof candidate === "string" && candidate.trim().length > 0
  );
};

const normalizePollingStatus = (raw: unknown): PollingStatusState => {
  let statusSource = raw;
  if (typeof statusSource === "string") {
    try {
      statusSource = JSON.parse(statusSource);
    } catch {
      statusSource = {};
    }
  }

  const status =
    statusSource && typeof statusSource === "object"
      ? (statusSource as Record<string, unknown>)
      : {};

  return {
    paused: pickBoolean(status.paused, status.Paused) ?? false,
    nextPollAt:
      pickNonEmptyString(
        status.nextPollAt,
        status.NextPollAt,
        status.next_poll_at
      ) ?? "",
  };
};

const extractEventPayload = (event: unknown): unknown => {
  if (event && typeof event === "object" && "data" in event) {
    return (event as { data?: unknown }).data;
  }
  return event;
};

const hasPollingStatusFields = (raw: unknown): boolean => {
  if (!raw || typeof raw !== "object") {
    return false;
  }
  const status = raw as Record<string, unknown>;
  return (
    "paused" in status ||
    "Paused" in status ||
    "nextPollAt" in status ||
    "NextPollAt" in status ||
    "next_poll_at" in status
  );
};

export function usePollingController({
  refreshJobs,
  refreshFilters,
  setAppError,
}: UsePollingControllerOptions): UsePollingControllerReturn {
  const [isPolling, setIsPolling] = useState(false);
  const [pollingPaused, setPollingPaused] = useState(false);
  const [nextPollAt, setNextPollAt] = useState("");
  const [pollRun, setPollRun] = useState<PollRunResult | null>(null);

  const predictNextPollAt = useCallback(() => {
    return new Date(Date.now() + defaultPollIntervalMs).toISOString();
  }, []);

  const applyPollingStatus = useCallback((status: PollingStatusState) => {
    setPollingPaused(status.paused);
    setNextPollAt((prev) => {
      if (status.nextPollAt) {
        return status.nextPollAt;
      }
      if (status.paused) {
        return "";
      }
      return prev;
    });
  }, []);

  const refreshPollingStatus = useCallback(
    async (surfaceError: boolean = false) => {
      try {
        const status = normalizePollingStatus(await GetPollingStatus());
        applyPollingStatus(status);
        return status;
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to load polling status:", message);
        if (surfaceError) {
          setAppError((prev) => prev ?? message);
        }
        return null;
      }
    },
    [applyPollingStatus, setAppError]
  );

  useEffect(() => {
    void refreshPollingStatus(true);
  }, [refreshPollingStatus]);

  // Keep header/status current when the app wakes or regains focus.
  useEffect(() => {
    let cancelled = false;

    const syncFromScheduler = async () => {
      if (cancelled) return;
      await Promise.all([refreshJobs(), refreshPollingStatus(false)]);
    };

    const syncIfVisible = () => {
      if (document.visibilityState !== "hidden") {
        void syncFromScheduler();
      }
    };

    const syncNow = () => {
      void syncFromScheduler();
    };

    const onPollingStatusChanged = (event: unknown) => {
      const payload = extractEventPayload(event);

      if (hasPollingStatusFields(payload)) {
        const status = normalizePollingStatus(payload);
        applyPollingStatus(status);
        if (!status.paused && !status.nextPollAt) {
          syncNow();
        }
        return;
      }

      syncNow();
    };

    const unsubscribe = Events.On(pollingStatusEvent, onPollingStatusChanged);
    const intervalId = window.setInterval(syncIfVisible, schedulerSyncIntervalMs);
    window.addEventListener("focus", syncNow);
    document.addEventListener("visibilitychange", syncIfVisible);

    return () => {
      cancelled = true;
      if (typeof unsubscribe === "function") {
        unsubscribe();
      }
      window.clearInterval(intervalId);
      window.removeEventListener("focus", syncNow);
      document.removeEventListener("visibilitychange", syncIfVisible);
    };
  }, [applyPollingStatus, refreshJobs, refreshPollingStatus]);

  const pollNow = useCallback(async () => {
    setIsPolling(true);
    setAppError(null);
    setPollRun(null);
    try {
      const run = await PollNow();
      if (run && (run.totalFilters > 0 || run.cycleError)) {
        setPollRun(run);
      }
      if (run?.diagnosticsError) {
        setAppError(run.diagnosticsError);
      }
      if (!pollingPaused) {
        setNextPollAt((prev) => prev || predictNextPollAt());
      }
      await Promise.all([
        refreshJobs(),
        refreshFilters(),
        refreshPollingStatus(true),
      ]);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Poll failed:", message);
      setAppError(message);
    } finally {
      setIsPolling(false);
    }
  }, [
    pollingPaused,
    predictNextPollAt,
    refreshFilters,
    refreshJobs,
    refreshPollingStatus,
    setAppError,
  ]);

  const togglePolling = useCallback(async () => {
    const newPaused = !pollingPaused;
    try {
      await SetPollingPaused(newPaused);
      setPollingPaused(newPaused);
      if (!newPaused) {
        setNextPollAt((prev) => prev || predictNextPollAt());
        await refreshPollingStatus(true);
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to toggle polling:", message);
      setAppError(message);
    }
  }, [pollingPaused, predictNextPollAt, refreshPollingStatus, setAppError]);

  const clearPollRun = useCallback(() => {
    setPollRun(null);
  }, []);

  return {
    isPolling,
    pollingPaused,
    nextPollAt,
    pollRun,
    pollNow,
    togglePolling,
    clearPollRun,
  };
}
