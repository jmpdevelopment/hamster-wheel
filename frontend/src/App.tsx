import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import {
  GetFirstRunComplete,
  GetKeyboardShortcuts,
  SetFirstRunComplete,
} from "../bindings/hamster-wheel/settingsservice";
import { Events } from "@wailsio/runtime";
import { useJobs } from "./hooks/useJobs";
import { useFilters } from "./hooks/useFilters";
import { useKeyboardShortcuts } from "./hooks/useKeyboardShortcuts";
import { usePollingController } from "./hooks/usePollingController";
import { Header } from "./components/Header";
import { ErrorBanner } from "./components/ErrorBanner";
import { FilterPanel } from "./components/FilterPanel";
import { JobList } from "./components/JobList";
import { JobDetail } from "./components/JobDetail";
import { PollResultToast } from "./components/PollResultToast";
import { SettingsPanel } from "./components/SettingsPanel";
import { Toast } from "./components/Toast";
import { ShortcutsHelp } from "./components/ShortcutsHelp";
import { useTheme } from "./hooks/useTheme";
import { FilterGroup, groupFilters } from "./lib/filterGroups";

function App() {
  const jobs = useJobs();
  const filters = useFilters();
  const { theme, setTheme } = useTheme();

  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [filterByFilterId, setFilterByFilterId] = useState<string | null>(null);
  const [appError, setAppError] = useState<string | null>(null);
  const [filteredJobIds, setFilteredJobIds] = useState<string[]>([]);
  const [keyboardShortcutsEnabled, setKeyboardShortcutsEnabled] =
    useState(true);
  const [setupWizardOpen, setSetupWizardOpen] = useState(false);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const polling = usePollingController({
    refreshJobs: jobs.refresh,
    refreshFilters: filters.refresh,
    setAppError,
  });
  const filterGroups = useMemo(
    () => groupFilters(filters.filters),
    [filters.filters]
  );

  // Fetch keyboard shortcuts setting on mount.
  useEffect(() => {
    let cancelled = false;

    const loadKeyboardShortcuts = async () => {
      try {
        const val = await GetKeyboardShortcuts();
        // Empty string means default (enabled).
        if (!cancelled && val === "false") {
          setKeyboardShortcutsEnabled(false);
        }
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to load keyboard shortcuts setting:", message);
        if (!cancelled) {
          setAppError((prev) => prev ?? message);
        }
      }
    };

    void loadKeyboardShortcuts();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    const loadInitialSetupState = async () => {
      try {
        const firstRunComplete = await GetFirstRunComplete();
        if (!cancelled) {
          setSetupWizardOpen(!firstRunComplete);
        }
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to load initial setup state:", message);
        if (!cancelled) {
          setAppError((prev) => prev ?? message);
        }
      }
    };

    void loadInitialSetupState();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let refreshTimer: number | null = null;
    const scheduleRefresh = () => {
      if (refreshTimer !== null) {
        return;
      }
      // Coalesce rapid status updates from matcher batches.
      refreshTimer = window.setTimeout(() => {
        refreshTimer = null;
        void jobs.refresh();
      }, 150);
    };

    const unsubscribe = Events.On("matching:status-changed", scheduleRefresh);
    return () => {
      if (refreshTimer !== null) {
        clearTimeout(refreshTimer);
      }
      unsubscribe();
    };
  }, [jobs.refresh]);

  // Combine errors from hooks and app-level errors.
  const error = appError || jobs.error || filters.error;

  const handleDismissError = () => {
    setAppError(null);
    jobs.clearError();
    filters.clearError();
  };

  const handleToggleFilter = useCallback(
    async (filterGroup: FilterGroup, enabled: boolean) => {
      for (const filter of filterGroup.Filters) {
        await filters.updateFilter(
          filter.ID,
          filter.Name,
          filter.Keywords,
          filter.Location,
          filter.Source,
          enabled
        );
      }
    },
    [filters]
  );

  const handleDeleteJob = useCallback(
    async (id: string) => {
      await jobs.deleteJob(id);
      if (selectedJobId === id) {
        setSelectedJobId(null);
      }
    },
    [jobs, selectedJobId]
  );

  const handleDeleteJobs = useCallback(
    async (ids: string[]) => {
      if (ids.length === 0) {
        return;
      }

      await jobs.deleteJobs(ids);
      if (selectedJobId && ids.includes(selectedJobId)) {
        setSelectedJobId(null);
      }
    },
    [jobs, selectedJobId]
  );

  const handleRecalculateJobs = useCallback(
    async (ids: string[]) => {
      if (ids.length === 0) {
        return;
      }
      await jobs.recalculateMatchScores(ids);
    },
    [jobs]
  );

  const selectedJob = selectedJobId
    ? jobs.jobs.find((j) => j.ID === selectedJobId) ?? null
    : null;

  const jobCountsByFilterId = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const job of jobs.jobs) {
      if (!job.FilterID) {
        continue;
      }
      counts[job.FilterID] = (counts[job.FilterID] ?? 0) + 1;
    }
    return counts;
  }, [jobs.jobs]);
  const jobCountsByFilterGroupId = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const filterGroup of filterGroups) {
      counts[filterGroup.ID] = filterGroup.FilterIDs.reduce(
        (sum, filterID) => sum + (jobCountsByFilterId[filterID] ?? 0),
        0
      );
    }
    return counts;
  }, [filterGroups, jobCountsByFilterId]);

  const handleSelectJob = useCallback(
    (id: string | null) => {
      setSelectedJobId(id);
      if (id) setSettingsOpen(false);
    },
    []
  );

  const handleCompleteInitialSetup = useCallback(() => {
    void (async () => {
      try {
        await SetFirstRunComplete(true);
        setSetupWizardOpen(false);
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to persist setup completion:", message);
        setAppError((prev) => prev ?? message);
      }
    })();
  }, []);

  const { pendingDeleteId, clearPendingDelete, shortcutsHelpOpen, closeShortcutsHelp } = useKeyboardShortcuts({
    enabled: keyboardShortcutsEnabled && !setupWizardOpen,
    filteredJobIds,
    selectedJobId,
    settingsOpen: settingsOpen || setupWizardOpen,
    jobDetailOpen: selectedJob !== null,
    onSelectJob: handleSelectJob,
    onCloseJobDetail: () => setSelectedJobId(null),
    onCloseSettings: () => {
      if (!setupWizardOpen) {
        setSettingsOpen(false);
      }
    },
    onOpenSettings: () => {
      if (!setupWizardOpen) {
        setSettingsOpen(true);
      }
    },
    onPollNow: polling.pollNow,
    onDeleteJob: handleDeleteJob,
    onFocusSearch: () => searchInputRef.current?.focus(),
    isPolling: polling.isPolling,
    canPoll:
      filterGroups.length > 0 &&
      filterGroups.some((f) => f.Enabled),
  });

  return (
    <div className="relative flex flex-col h-screen bg-hw-bg text-hw-text">
      <Header
        onPollNow={polling.pollNow}
        isPolling={polling.isPolling}
        pollingPaused={polling.pollingPaused}
        nextPollAt={polling.nextPollAt}
        hasFilters={filterGroups.length > 0}
        hasEnabledFilters={filterGroups.some((f) => f.Enabled)}
        onOpenSettings={() => {
          if (!setupWizardOpen) {
            setSettingsOpen(true);
          }
        }}
      />

      <ErrorBanner message={error} onDismiss={handleDismissError} />

      <div className="flex flex-1 overflow-hidden relative">
        {/* Left: Filters */}
        <FilterPanel
          filters={filterGroups}
          jobCountsByFilterId={jobCountsByFilterGroupId}
          loading={filters.loading}
          onCreateFilter={async (name, keywords, location, sources) => {
            const createdFilterIDs: string[] = [];
            try {
              for (const source of sources) {
                const id = await filters.createFilter(
                  name,
                  keywords,
                  location,
                  source
                );
                createdFilterIDs.push(id);
              }
            } catch (err: unknown) {
              for (const id of createdFilterIDs) {
                await filters.deleteFilter(id).catch(() => {
                  // Hook tracks any rollback failures through error state.
                });
              }
              throw err;
            }
          }}
          onToggleFilter={handleToggleFilter}
          onDeleteFilter={async (filterGroup, deleteAssociatedJobs) => {
            const filterIDSet = new Set(filterGroup.FilterIDs);
            if (deleteAssociatedJobs) {
              const associatedJobIDs = jobs.jobs
                .filter(
                  (job) => job.FilterID !== null && filterIDSet.has(job.FilterID)
                )
                .map((job) => job.ID);
              await handleDeleteJobs(associatedJobIDs);
            }
            for (const filterID of filterGroup.FilterIDs) {
              await filters.deleteFilter(filterID);
            }
          }}
        />

        {/* Center: Job List */}
        <div className="flex-1 min-w-0 border-r border-hw-border">
          <JobList
            jobs={jobs.jobs}
            filters={filterGroups}
            filtersLoading={filters.loading}
            loading={jobs.loading}
            selectedJobId={selectedJobId}
            onSelectJob={(id) => handleSelectJob(id)}
            filterByFilterId={filterByFilterId}
            onFilterChange={setFilterByFilterId}
            onFilteredJobsChange={setFilteredJobIds}
            searchInputRef={searchInputRef}
            onSetFavoriteJobs={jobs.setJobsFavorite}
            onToggleFavoriteJob={async (jobID) => {
              const job = jobs.jobs.find((candidate) => candidate.ID === jobID);
              if (!job) {
                return;
              }
              await jobs.setJobFavorite(jobID, !job.IsFavorite);
            }}
            onDeleteJobs={handleDeleteJobs}
            onRecalculateJobs={handleRecalculateJobs}
          />
        </div>

        {/* Right: Job Detail */}
        {selectedJob && (
          <div className="w-[400px] shrink-0">
            <JobDetail
              job={selectedJob}
              onDelete={handleDeleteJob}
              onClose={() => setSelectedJobId(null)}
              onRefresh={jobs.refresh}
            />
          </div>
        )}

        {/* Settings Panel (overlays right side) */}
        {settingsOpen && !setupWizardOpen && (
          <SettingsPanel
            onClose={() => setSettingsOpen(false)}
            theme={theme}
            onSetTheme={setTheme}
            onError={setAppError}
            keyboardShortcuts={keyboardShortcutsEnabled}
            onSetKeyboardShortcuts={setKeyboardShortcutsEnabled}
          />
        )}
      </div>

      {setupWizardOpen && (
        <div className="absolute inset-0 z-20 bg-black/45 p-4 flex items-center justify-center">
          <SettingsPanel
            onClose={() => {}}
            mode="wizard"
            onCompleteSetup={handleCompleteInitialSetup}
            theme={theme}
            onSetTheme={setTheme}
            onError={setAppError}
            keyboardShortcuts={keyboardShortcutsEnabled}
            onSetKeyboardShortcuts={setKeyboardShortcutsEnabled}
          />
        </div>
      )}

      <PollResultToast
        run={polling.pollRun}
        onDismiss={polling.clearPollRun}
      />

      {shortcutsHelpOpen && (
        <ShortcutsHelp onClose={closeShortcutsHelp} />
      )}

      {pendingDeleteId && (
        <Toast
          variant="info"
          title="Press Delete again to confirm"
          duration={0}
          onDismiss={clearPendingDelete}
        >
          Press Escape to cancel.
        </Toast>
      )}
    </div>
  );
}

export default App;
