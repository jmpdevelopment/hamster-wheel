import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FixedSizeList, ListChildComponentProps } from "react-window";
import { AutoSizer } from "react-virtualized-auto-sizer";
import {
  Job,
  SearchFilter,
} from "../../bindings/hamster-wheel/internal/db/models";
import {
  GetJobListPreferences,
  SetJobListPreferences,
} from "../../bindings/hamster-wheel/settingsservice";
import { JobCard } from "./JobCard";
import { SearchInput } from "./SearchInput";
import { EmptyState } from "./EmptyState";
import {
  useJobSearch,
  type JobSortMode,
  type PostedDateFilterMode,
  type MatchScoreFilterMode,
} from "../hooks/useJobSearch";
import { Button } from "./Button";
import { ConfirmAction } from "./ConfirmAction";

const ITEM_HEIGHT = 74;

const sortOptions: Array<{ value: JobSortMode; label: string }> = [
  { value: "posted-desc", label: "Posted date: newest first" },
  { value: "posted-asc", label: "Posted date: oldest first" },
  { value: "score-desc", label: "Match score: highest first" },
  { value: "score-asc", label: "Match score: lowest first" },
];

const postedDateFilterOptions: Array<{
  value: PostedDateFilterMode;
  label: string;
}> = [
  { value: "any", label: "Any posting date" },
  { value: "last-24h", label: "Posted in last 24 hours" },
  { value: "last-7d", label: "Posted in last 7 days" },
  { value: "last-30d", label: "Posted in last 30 days" },
];

const matchScoreFilterOptions: Array<{
  value: MatchScoreFilterMode;
  label: string;
}> = [
  { value: "any", label: "Any match score" },
  { value: "scored", label: "Scored jobs only" },
  { value: "score-80", label: "Match score 80%+" },
  { value: "score-60", label: "Match score 60%+" },
  { value: "score-40", label: "Match score 40%+" },
];

function isJobSortMode(value: string): value is JobSortMode {
  return sortOptions.some((option) => option.value === value);
}

function isPostedDateFilterMode(value: string): value is PostedDateFilterMode {
  return postedDateFilterOptions.some((option) => option.value === value);
}

function isMatchScoreFilterMode(value: string): value is MatchScoreFilterMode {
  return matchScoreFilterOptions.some((option) => option.value === value);
}

interface JobListProps {
  jobs: Job[];
  filters: SearchFilter[];
  loading: boolean;
  selectedJobId: string | null;
  onSelectJob: (id: string) => void;
  filterByFilterId: string | null;
  onFilterChange: (filterId: string | null) => void;
  onFilteredJobsChange?: (jobIds: string[]) => void;
  searchInputRef?: React.Ref<HTMLInputElement>;
  onSetFavoriteJobs: (jobIds: string[], favorite: boolean) => Promise<void>;
  onToggleFavoriteJob: (jobId: string) => Promise<void>;
  onDeleteJobs: (jobIds: string[]) => Promise<void>;
}

export function JobList({
  jobs,
  filters,
  loading,
  selectedJobId,
  onSelectJob,
  filterByFilterId,
  onFilterChange,
  onFilteredJobsChange,
  searchInputRef,
  onSetFavoriteJobs,
  onToggleFavoriteJob,
  onDeleteJobs,
}: JobListProps) {
  const {
    searchTerm,
    setSearchTerm,
    sortMode,
    setSortMode,
    postedDateFilterMode,
    setPostedDateFilterMode,
    matchScoreFilterMode,
    setMatchScoreFilterMode,
    filteredJobs,
  } = useJobSearch(jobs, filterByFilterId);
  const [showFavoritesOnly, setShowFavoritesOnly] = useState(false);
  const [selectedJobIds, setSelectedJobIds] = useState<Set<string>>(
    () => new Set()
  );
  const preferencesHydratedRef = useRef(false);
  const listRef = useRef<FixedSizeList>(null);
  const selectAllRef = useRef<HTMLInputElement>(null);
  const lastToggledJobIDRef = useRef<string | null>(null);
  const shiftPressedRef = useRef(false);

  const visibleJobs = useMemo(
    () =>
      showFavoritesOnly
        ? filteredJobs.filter((job) => job.IsFavorite)
        : filteredJobs,
    [filteredJobs, showFavoritesOnly]
  );

  const hasSearch = searchTerm.trim().length > 0;
  const hasFilter =
    filterByFilterId !== null ||
    postedDateFilterMode !== "any" ||
    matchScoreFilterMode !== "any";
  const visibleJobIndexByID = useMemo(() => {
    const indexByID = new Map<string, number>();
    visibleJobs.forEach((job, index) => {
      indexByID.set(job.ID, index);
    });
    return indexByID;
  }, [visibleJobs]);
  const selectedIDs = useMemo(
    () => Array.from(selectedJobIds),
    [selectedJobIds]
  );
  const selectedCount = selectedIDs.length;
  const selectedVisibleCount = useMemo(
    () => visibleJobs.filter((job) => selectedJobIds.has(job.ID)).length,
    [visibleJobs, selectedJobIds]
  );
  const allVisibleSelected =
    visibleJobs.length > 0 && selectedVisibleCount === visibleJobs.length;
  const hasVisibleSelection =
    selectedVisibleCount > 0 && selectedVisibleCount < visibleJobs.length;

  useEffect(() => {
    if (selectAllRef.current) {
      selectAllRef.current.indeterminate = hasVisibleSelection;
    }
  }, [hasVisibleSelection]);

  useEffect(() => {
    let cancelled = false;

    const loadPreferences = async () => {
      try {
        const preferences = await GetJobListPreferences();
        if (cancelled || !preferences) {
          return;
        }

        const filterByFilterID = preferences.filterByFilterId ?? "";
        onFilterChange(filterByFilterID || null);

        if (isJobSortMode(preferences.sortMode)) {
          setSortMode(preferences.sortMode);
        }
        if (isPostedDateFilterMode(preferences.postedDateFilterMode)) {
          setPostedDateFilterMode(preferences.postedDateFilterMode);
        }
        if (isMatchScoreFilterMode(preferences.matchScoreFilterMode)) {
          setMatchScoreFilterMode(preferences.matchScoreFilterMode);
        }
        setShowFavoritesOnly(preferences.showFavoritesOnly === true);
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to load job list preferences:", message);
      } finally {
        preferencesHydratedRef.current = true;
      }
    };

    void loadPreferences();
    return () => {
      cancelled = true;
    };
  }, [
    onFilterChange,
    setSortMode,
    setPostedDateFilterMode,
    setMatchScoreFilterMode,
  ]);

  useEffect(() => {
    if (!preferencesHydratedRef.current) {
      return;
    }

    void SetJobListPreferences({
      filterByFilterId: filterByFilterId ?? "",
      sortMode,
      postedDateFilterMode,
      matchScoreFilterMode,
      showFavoritesOnly,
    }).catch((err: unknown) => {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save job list preferences:", message);
    });
  }, [
    filterByFilterId,
    sortMode,
    postedDateFilterMode,
    matchScoreFilterMode,
    showFavoritesOnly,
  ]);

  useEffect(() => {
    const handleKeyDown = (event: Event) => {
      if (event instanceof KeyboardEvent && event.key === "Shift") {
        shiftPressedRef.current = true;
      }
    };
    const handleKeyUp = (event: Event) => {
      if (event instanceof KeyboardEvent && event.key === "Shift") {
        shiftPressedRef.current = false;
      }
    };
    const handleWindowBlur = () => {
      shiftPressedRef.current = false;
    };

    document.addEventListener("keydown", handleKeyDown, true);
    document.addEventListener("keyup", handleKeyUp, true);
    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);
    window.addEventListener("blur", handleWindowBlur);
    return () => {
      document.removeEventListener("keydown", handleKeyDown, true);
      document.removeEventListener("keyup", handleKeyUp, true);
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
      window.removeEventListener("blur", handleWindowBlur);
    };
  }, []);

  // Notify parent when visible jobs change.
  useEffect(() => {
    onFilteredJobsChange?.(visibleJobs.map((job) => job.ID));
  }, [visibleJobs, onFilteredJobsChange]);

  // Keep selected IDs scoped to currently visible rows.
  useEffect(() => {
    const visibleIDSet = new Set(visibleJobs.map((job) => job.ID));
    setSelectedJobIds((previous) => {
      let changed = false;
      const next = new Set<string>();
      for (const id of previous) {
        if (visibleIDSet.has(id)) {
          next.add(id);
        } else {
          changed = true;
        }
      }
      return changed ? next : previous;
    });
  }, [visibleJobs]);

  // Scroll to selected job when selection changes.
  useEffect(() => {
    if (!selectedJobId || !listRef.current) return;
    const index = visibleJobs.findIndex((j) => j.ID === selectedJobId);
    if (index >= 0) {
      listRef.current.scrollToItem(index, "smart");
    }
  }, [selectedJobId, visibleJobs]);

  const applySelectionRange = useCallback(
    (next: Set<string>, anchorJobID: string | null, currentIndex: number) => {
      const anchorIndex =
        anchorJobID === null ? undefined : visibleJobIndexByID.get(anchorJobID);
      const canApplyRange =
        anchorIndex !== undefined &&
        anchorIndex >= 0 &&
        anchorIndex < visibleJobs.length;

      if (!canApplyRange) {
        return false;
      }

      const start = Math.min(anchorIndex, currentIndex);
      const end = Math.max(anchorIndex, currentIndex);
      for (let index = start; index <= end; index += 1) {
        next.add(visibleJobs[index].ID);
      }
      return true;
    },
    [visibleJobIndexByID, visibleJobs]
  );

  const handleToggleJobSelection = useCallback(
    (jobID: string, checked: boolean, shiftKey: boolean) => {
      const currentIndex = visibleJobIndexByID.get(jobID);
      if (currentIndex === undefined) {
        return;
      }
      const rangeRequested = shiftKey || shiftPressedRef.current;
      const anchorJobIDAtClick = lastToggledJobIDRef.current ?? selectedJobId;

      setSelectedJobIds((previous) => {
        const next = new Set(previous);
        if (rangeRequested && applySelectionRange(next, anchorJobIDAtClick, currentIndex)) {
          return next;
        }

        if (checked) {
          next.add(jobID);
        } else {
          next.delete(jobID);
        }
        return next;
      });

      lastToggledJobIDRef.current = jobID;
    },
    [applySelectionRange, selectedJobId, visibleJobIndexByID]
  );

  const handleRowClick = useCallback(
    (jobID: string, shiftKey: boolean) => {
      const currentIndex = visibleJobIndexByID.get(jobID);
      if (currentIndex === undefined) {
        onSelectJob(jobID);
        return;
      }
      const rangeRequested = shiftKey || shiftPressedRef.current;
      const anchorJobIDAtClick = lastToggledJobIDRef.current ?? selectedJobId;

      if (rangeRequested) {
        setSelectedJobIds((previous) => {
          const next = new Set(previous);
          if (!applySelectionRange(next, anchorJobIDAtClick, currentIndex)) {
            next.add(jobID);
          }

          return next;
        });
      }

      lastToggledJobIDRef.current = jobID;
      onSelectJob(jobID);
    },
    [applySelectionRange, onSelectJob, selectedJobId, visibleJobIndexByID]
  );

  const handleSelectAllVisible = useCallback(
    (checked: boolean) => {
      if (!checked) {
        setSelectedJobIds(new Set());
        lastToggledJobIDRef.current = null;
        return;
      }
      setSelectedJobIds(new Set(visibleJobs.map((job) => job.ID)));
      lastToggledJobIDRef.current = null;
    },
    [visibleJobs]
  );

  const handleDeleteSelected = useCallback(async () => {
    if (selectedIDs.length === 0) {
      return;
    }
    await onDeleteJobs(selectedIDs);
    setSelectedJobIds(new Set());
    lastToggledJobIDRef.current = null;
  }, [onDeleteJobs, selectedIDs]);

  const handleFavoriteSelected = useCallback(
    (favorite: boolean) => {
      if (selectedIDs.length === 0) {
        return;
      }
      void Promise.resolve(onSetFavoriteJobs(selectedIDs, favorite)).catch(() => {
        // Parent tracks mutation errors for display.
      });
    },
    [onSetFavoriteJobs, selectedIDs]
  );

  const Row = useCallback(
    ({ index, style }: ListChildComponentProps) => {
      const job = visibleJobs[index];
      return (
        <JobCard
          key={job.ID}
          job={job}
          isSelected={job.ID === selectedJobId}
          isChecked={selectedJobIds.has(job.ID)}
          isFavorite={job.IsFavorite}
          onClick={(shiftKey) => handleRowClick(job.ID, shiftKey)}
          onToggleChecked={(checked, shiftKey) =>
            handleToggleJobSelection(job.ID, checked, shiftKey)
          }
          onToggleFavorite={() => {
            void Promise.resolve(onToggleFavoriteJob(job.ID)).catch(() => {
              // Parent tracks mutation errors for display.
            });
          }}
          style={style}
        />
      );
    },
    [
      visibleJobs,
      selectedJobId,
      selectedJobIds,
      handleRowClick,
      handleToggleJobSelection,
      onToggleFavoriteJob,
    ]
  );

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header: Search + Filter + Actions */}
      <div className="shrink-0 px-3 py-2 border-b border-hw-border space-y-2">
        <SearchInput
          ref={searchInputRef}
          value={searchTerm}
          onChange={setSearchTerm}
        />
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
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
          <select
            value={sortMode}
            onChange={(event) => setSortMode(event.target.value as JobSortMode)}
            className="w-full px-2 py-1.5 text-sm rounded bg-hw-bg border border-hw-border text-hw-text focus:outline-none focus:border-hw-accent"
            aria-label="Sort jobs"
          >
            {sortOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>

          <select
            value={postedDateFilterMode}
            onChange={(event) =>
              setPostedDateFilterMode(
                event.target.value as PostedDateFilterMode
              )
            }
            className="w-full px-2 py-1.5 text-sm rounded bg-hw-bg border border-hw-border text-hw-text focus:outline-none focus:border-hw-accent"
            aria-label="Filter jobs by posting date"
          >
            {postedDateFilterOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>

          <select
            value={matchScoreFilterMode}
            onChange={(event) =>
              setMatchScoreFilterMode(
                event.target.value as MatchScoreFilterMode
              )
            }
            className="w-full px-2 py-1.5 text-sm rounded bg-hw-bg border border-hw-border text-hw-text focus:outline-none focus:border-hw-accent"
            aria-label="Filter jobs by match score"
          >
            {matchScoreFilterOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>

        {!loading && jobs.length > 0 && (
          <div className="flex flex-wrap items-center gap-2 pt-1">
            <label className="inline-flex items-center gap-2 text-xs text-hw-text-muted">
              <input
                ref={selectAllRef}
                type="checkbox"
                checked={allVisibleSelected}
                onChange={(event) => handleSelectAllVisible(event.target.checked)}
                className="h-4 w-4 rounded border-hw-border bg-hw-bg text-hw-accent focus:ring-hw-accent"
                aria-label="Select all visible jobs"
              />
              {selectedCount} selected
            </label>

            <Button
              variant="secondary"
              size="sm"
              onClick={() => handleFavoriteSelected(true)}
              disabled={selectedCount === 0}
            >
              Favorite
            </Button>

            <Button
              variant="secondary"
              size="sm"
              onClick={() => handleFavoriteSelected(false)}
              disabled={selectedCount === 0}
            >
              Unfavorite
            </Button>

            <ConfirmAction
              onConfirm={() => {
                void handleDeleteSelected();
              }}
              confirmLabel={
                selectedCount <= 1 ? "Delete job" : `Delete ${selectedCount} jobs`
              }
            >
              <Button variant="danger" size="sm" disabled={selectedCount === 0}>
                Delete
              </Button>
            </ConfirmAction>

            <Button
              variant={showFavoritesOnly ? "primary" : "ghost"}
              size="sm"
              onClick={() => setShowFavoritesOnly((previous) => !previous)}
              aria-label={
                showFavoritesOnly
                  ? "Show all jobs"
                  : "Show only favorite jobs"
              }
            >
              {showFavoritesOnly ? "Show all" : "Favorites only"}
            </Button>
            <p className="text-xs text-hw-text-muted">
            {visibleJobs.length === jobs.length && !showFavoritesOnly
              ? `${jobs.length} jobs`
              : `${visibleJobs.length} of ${jobs.length} jobs`}
          </p>
          </div>
        )}
      </div>

      {/* Job list (virtualized) */}
      <div className="flex-1">
        {loading ? (
          <p className="text-sm text-hw-text-muted px-3 py-8 text-center">
            Loading...
          </p>
        ) : visibleJobs.length === 0 ? (
          <EmptyState
            title={
              hasSearch || hasFilter
                ? "No matching jobs"
                : showFavoritesOnly
                  ? "No favorite jobs"
                  : "No jobs yet"
            }
            description={
              hasSearch || hasFilter
                ? "Try adjusting your search or filter."
                : showFavoritesOnly
                  ? "Mark jobs as favorites to quickly return to them."
                  : "Make sure you have enabled filters and try polling."
            }
          />
        ) : (
          <AutoSizer
            renderProp={({ height, width }) =>
              height && width ? (
                <FixedSizeList
                  ref={listRef}
                  height={height}
                  width={width}
                  itemCount={visibleJobs.length}
                  itemSize={ITEM_HEIGHT}
                  itemKey={(index) => visibleJobs[index].ID}
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
