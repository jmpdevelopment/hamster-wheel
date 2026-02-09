import { useState, useCallback, useEffect } from "react";
import {
  PollNow,
  GetPollingStatus,
  SetPollingPaused,
} from "../bindings/hamster-wheel/appservice";
import { PollResult } from "../bindings/hamster-wheel/internal/scheduler/models";
import { useJobs } from "./hooks/useJobs";
import { useFilters } from "./hooks/useFilters";
import { Header } from "./components/Header";
import { ErrorBanner } from "./components/ErrorBanner";
import { FilterPanel } from "./components/FilterPanel";
import { JobList } from "./components/JobList";
import { JobDetail } from "./components/JobDetail";
import { PollResultToast } from "./components/PollResultToast";

function App() {
  const jobs = useJobs();
  const filters = useFilters();

  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [filterByFilterId, setFilterByFilterId] = useState<string | null>(null);
  const [isPolling, setIsPolling] = useState(false);
  const [pollResults, setPollResults] = useState<PollResult[] | null>(
    null
  );
  const [appError, setAppError] = useState<string | null>(null);
  const [pollingPaused, setPollingPaused] = useState(false);
  const [nextPollAt, setNextPollAt] = useState("");

  // Fetch polling status on mount.
  useEffect(() => {
    GetPollingStatus().then((status) => {
      setPollingPaused(status.paused);
      setNextPollAt(status.nextPollAt);
    });
  }, []);

  // Combine errors from hooks and app-level errors.
  const error = appError || jobs.error || filters.error;

  const handleDismissError = () => {
    setAppError(null);
  };

  const handlePollNow = useCallback(async () => {
    setIsPolling(true);
    setAppError(null);
    try {
      const results = await PollNow();
      setPollResults(results ?? []);
      // Refresh data and polling status after polling.
      const [, , status] = await Promise.all([
        jobs.refresh(),
        filters.refresh(),
        GetPollingStatus(),
      ]);
      setNextPollAt(status.nextPollAt);
      setPollingPaused(status.paused);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Poll failed:", message);
      setAppError(message);
    } finally {
      setIsPolling(false);
    }
  }, [jobs, filters]);

  const handleTogglePolling = useCallback(async () => {
    const newPaused = !pollingPaused;
    await SetPollingPaused(newPaused);
    setPollingPaused(newPaused);
    if (!newPaused) {
      const status = await GetPollingStatus();
      setNextPollAt(status.nextPollAt);
    }
  }, [pollingPaused]);

  const handleToggleFilter = useCallback(
    (filter: { ID: string; Name: string; Keywords: string; Location: string; Source: string }, enabled: boolean) => {
      filters.updateFilter(
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

  const selectedJob = selectedJobId
    ? jobs.jobs.find((j) => j.ID === selectedJobId) ?? null
    : null;

  return (
    <div className="flex flex-col h-screen bg-hw-bg text-hw-text">
      <Header
        jobCount={jobs.jobCount}
        onPollNow={handlePollNow}
        isPolling={isPolling}
        pollingPaused={pollingPaused}
        nextPollAt={nextPollAt}
        onTogglePolling={handleTogglePolling}
      />

      <ErrorBanner message={error} onDismiss={handleDismissError} />

      <div className="flex flex-1 overflow-hidden">
        {/* Left: Filters */}
        <FilterPanel
          filters={filters.filters}
          loading={filters.loading}
          onCreateFilter={async (name, keywords, location, source) => {
            await filters.createFilter(name, keywords, location, source);
          }}
          onToggleFilter={handleToggleFilter}
          onDeleteFilter={filters.deleteFilter}
          onError={setAppError}
        />

        {/* Center: Job List */}
        <div className="flex-1 min-w-0 border-r border-hw-border">
          <JobList
            jobs={jobs.jobs}
            filters={filters.filters}
            loading={jobs.loading}
            selectedJobId={selectedJobId}
            onSelectJob={setSelectedJobId}
            filterByFilterId={filterByFilterId}
            onFilterChange={setFilterByFilterId}
          />
        </div>

        {/* Right: Job Detail */}
        {selectedJob && (
          <div className="w-[400px] shrink-0">
            <JobDetail
              job={selectedJob}
              onDelete={handleDeleteJob}
              onClose={() => setSelectedJobId(null)}
            />
          </div>
        )}
      </div>

      <PollResultToast
        results={pollResults}
        onDismiss={() => setPollResults(null)}
      />
    </div>
  );
}

export default App;
