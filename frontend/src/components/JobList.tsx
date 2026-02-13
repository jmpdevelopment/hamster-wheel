import { useCallback } from "react";
import { FixedSizeList, ListChildComponentProps } from "react-window";
import { AutoSizer } from "react-virtualized-auto-sizer";
import {
  Job,
  SearchFilter,
} from "../../bindings/hamster-wheel/internal/db/models";
import { JobCard } from "./JobCard";
import { SearchInput } from "./SearchInput";
import { EmptyState } from "./EmptyState";
import { useJobSearch } from "../hooks/useJobSearch";

const ITEM_HEIGHT = 74;

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
  const { searchTerm, setSearchTerm, filteredJobs } = useJobSearch(
    jobs,
    filterByFilterId
  );

  const hasSearch = searchTerm.trim().length > 0;
  const hasFilter = filterByFilterId !== null;

  const Row = useCallback(
    ({ index, style }: ListChildComponentProps) => {
      const job = filteredJobs[index];
      return (
        <JobCard
          key={job.ID}
          job={job}
          isSelected={job.ID === selectedJobId}
          onClick={() => onSelectJob(job.ID)}
          style={style}
        />
      );
    },
    [filteredJobs, selectedJobId, onSelectJob]
  );

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header: Search + Filter */}
      <div className="shrink-0 px-3 py-2 border-b border-hw-border space-y-2">
        <SearchInput value={searchTerm} onChange={setSearchTerm} />
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
        {!loading && jobs.length > 0 && (
          <p className="text-xs text-hw-text-muted">
            {filteredJobs.length === jobs.length
              ? `${jobs.length} jobs`
              : `${filteredJobs.length} of ${jobs.length} jobs`}
          </p>
        )}
      </div>

      {/* Job list (virtualized) */}
      <div className="flex-1">
        {loading ? (
          <p className="text-sm text-hw-text-muted px-3 py-8 text-center">
            Loading...
          </p>
        ) : filteredJobs.length === 0 ? (
          <EmptyState
            title={hasSearch || hasFilter ? "No matching jobs" : "No jobs yet"}
            description={
              hasSearch || hasFilter
                ? "Try adjusting your search or filter."
                : "Make sure you have enabled filters and try polling."
            }
          />
        ) : (
          <AutoSizer
            renderProp={({ height, width }) =>
              height && width ? (
                <FixedSizeList
                  height={height}
                  width={width}
                  itemCount={filteredJobs.length}
                  itemSize={ITEM_HEIGHT}
                  itemKey={(index) => filteredJobs[index].ID}
                  overscanCount={5}
                >
                  {Row}
                </FixedSizeList>
              ) : null
            }
          />
        )}
      </div>
    </div>
  );
}
