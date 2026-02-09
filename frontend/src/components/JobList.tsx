import { Job, SearchFilter } from "../../bindings/hamster-wheel/internal/db/models";
import { JobCard } from "./JobCard";
import { EmptyState } from "./EmptyState";

interface JobListProps {
  jobs: Job[];
  filters: SearchFilter[];
  loading: boolean;
  selectedJobId: string | null;
  onSelectJob: (id: string) => void;
  filterByFilterId: string | null;
  onFilterChange: (filterId: string | null) => void;
}

export function JobList({
  jobs,
  filters,
  loading,
  selectedJobId,
  onSelectJob,
  filterByFilterId,
  onFilterChange,
}: JobListProps) {
  const filteredJobs = filterByFilterId
    ? jobs.filter((j) => j.FilterID === filterByFilterId)
    : jobs;

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Filter dropdown */}
      <div className="shrink-0 px-3 py-2 border-b border-hw-border">
        <select
          value={filterByFilterId ?? ""}
          onChange={(e) => onFilterChange(e.target.value || null)}
          className="w-full px-2 py-1.5 text-sm rounded bg-hw-bg border border-hw-border text-hw-text focus:outline-none focus:border-hw-accent"
          aria-label="Filter jobs by search filter"
        >
          <option value="">All Filters</option>
          {filters.map((f) => (
            <option key={f.ID} value={f.ID}>
              {f.Name}
            </option>
          ))}
        </select>
      </div>

      {/* Job list */}
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <p className="text-sm text-hw-text-muted px-3 py-8 text-center">
            Loading...
          </p>
        ) : filteredJobs.length === 0 ? (
          <EmptyState
            title="No jobs yet"
            description="Make sure you have enabled filters and try polling."
          />
        ) : (
          filteredJobs.map((job) => (
            <JobCard
              key={job.ID}
              job={job}
              isSelected={job.ID === selectedJobId}
              onClick={() => onSelectJob(job.ID)}
            />
          ))
        )}
      </div>
    </div>
  );
}
