import { useState } from "react";
import { SearchFilter } from "../../bindings/hamster-wheel/internal/db/models";
import { FilterCard } from "./FilterCard";
import { CreateFilterForm } from "./CreateFilterForm";
import { EmptyState } from "./EmptyState";

interface FilterPanelProps {
  filters: SearchFilter[];
  jobCountsByFilterId: Record<string, number>;
  loading: boolean;
  onCreateFilter: (
    name: string,
    keywords: string,
    location: string,
    source: string
  ) => Promise<void>;
  onToggleFilter: (filter: SearchFilter, enabled: boolean) => Promise<void>;
  onDeleteFilter: (
    id: string,
    deleteAssociatedJobs: boolean
  ) => Promise<void>;
}

export function FilterPanel({
  filters,
  jobCountsByFilterId,
  loading,
  onCreateFilter,
  onToggleFilter,
  onDeleteFilter,
}: FilterPanelProps) {
  const [showForm, setShowForm] = useState(false);

  const handleCreate = async (
    name: string,
    keywords: string,
    location: string,
    source: string
  ) => {
    await onCreateFilter(name, keywords, location, source);
    setShowForm(false);
  };

  return (
    <aside className="w-64 shrink-0 border-r border-hw-border bg-hw-bg overflow-y-auto">
      <div className="flex items-center justify-between px-3 py-2 border-b border-hw-border">
        <h2 className="text-sm font-semibold text-hw-text">Filters</h2>
        {!showForm && (
          <button
            onClick={() => setShowForm(true)}
            className="text-hw-accent hover:text-hw-accent-hover text-lg leading-none"
            aria-label="Add filter"
          >
            +
          </button>
        )}
      </div>

      <div className="p-2 space-y-2">
        {showForm && (
          <CreateFilterForm
            onSubmit={handleCreate}
            onCancel={() => setShowForm(false)}
          />
        )}

        {loading ? (
          <p className="text-sm text-hw-text-muted px-1 py-4 text-center">
            Loading...
          </p>
        ) : filters.length === 0 ? (
          <EmptyState
            title="No filters"
            description="Create a search filter to start finding jobs."
          />
        ) : (
          filters.map((filter) => (
            <FilterCard
              key={filter.ID}
              filter={filter}
              associatedJobCount={jobCountsByFilterId[filter.ID] ?? 0}
              onToggle={(enabled) => onToggleFilter(filter, enabled)}
              onDelete={(deleteAssociatedJobs) =>
                onDeleteFilter(filter.ID, deleteAssociatedJobs)
              }
            />
          ))
        )}
      </div>
    </aside>
  );
}
