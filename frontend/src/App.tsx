import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import { GetKeyboardShortcuts } from "../bindings/hamster-wheel/settingsservice";
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
  const searchInputRef = useRef<HTMLInputElement>(null);
  const polling = usePollingController({
    refreshJobs: jobs.refresh,
    refreshFilters: filters.refresh,
    setAppError,
  });

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

  // Combine errors from hooks and app-level errors.
  const error = appError || jobs.error || filters.error;

  const handleDismissError = () => {
    setAppError(null);
    jobs.clearError();
    filters.clearError();
  };

  const handleToggleFilter = useCallback(
    async (
      filter: {
        ID: string;
        Name: string;
        Keywords: string;
        Location: string;
        Source: string;
      },
      enabled: boolean
    ) => {
      await filters.updateFilter(
        filter.ID,
        filter.Name,
        filter.Keywords,
        filter.Location,
        filter.Source,
        enabled
      );
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

  const handleSelectJob = useCallback(
    (id: string | null) => {
      setSelectedJobId(id);
      if (id) setSettingsOpen(false);
    },
    []
  );

  const { pendingDeleteId, clearPendingDelete, shortcutsHelpOpen, closeShortcutsHelp } = useKeyboardShortcuts({
    enabled: keyboardShortcutsEnabled,
    filteredJobIds,
    selectedJobId,
    settingsOpen,
    jobDetailOpen: selectedJob !== null,
    onSelectJob: handleSelectJob,
    onCloseJobDetail: () => setSelectedJobId(null),
    onCloseSettings: () => setSettingsOpen(false),
    onOpenSettings: () => setSettingsOpen(true),
    onPollNow: polling.pollNow,
    onDeleteJob: handleDeleteJob,
    onFocusSearch: () => searchInputRef.current?.focus(),
    isPolling: polling.isPolling,
    canPoll:
      filters.filters.length > 0 &&
      filters.filters.some((f) => f.Enabled),
  });

  return (
    <div className="flex flex-col h-screen bg-hw-bg text-hw-text">
      <Header
        jobCount={jobs.jobCount}
        onPollNow={polling.pollNow}
        isPolling={polling.isPolling}
        pollingPaused={polling.pollingPaused}
        nextPollAt={polling.nextPollAt}
        onTogglePolling={polling.togglePolling}
        hasFilters={filters.filters.length > 0}
        hasEnabledFilters={filters.filters.some((f) => f.Enabled)}
        onOpenSettings={() => setSettingsOpen(true)}
      />

      <ErrorBanner message={error} onDismiss={handleDismissError} />

      <div className="flex flex-1 overflow-hidden relative">
        {/* Left: Filters */}
        <FilterPanel
          filters={filters.filters}
          jobCountsByFilterId={jobCountsByFilterId}
          loading={filters.loading}
          onCreateFilter={async (name, keywords, location, source) => {
            await filters.createFilter(name, keywords, location, source);
          }}
          onToggleFilter={handleToggleFilter}
          onDeleteFilter={async (id, deleteAssociatedJobs) => {
            if (deleteAssociatedJobs) {
              const associatedJobIDs = jobs.jobs
                .filter((job) => job.FilterID === id)
                .map((job) => job.ID);
              await handleDeleteJobs(associatedJobIDs);
            }
            await filters.deleteFilter(id);
          }}
        />

        {/* Center: Job List */}
        <div className="flex-1 min-w-0 border-r border-hw-border">
          <JobList
            jobs={jobs.jobs}
            filters={filters.filters}
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
        {settingsOpen && (
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
