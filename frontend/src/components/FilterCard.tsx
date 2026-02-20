import { useState } from "react";
import { FilterGroup } from "../lib/filterGroups";
import { Button } from "./Button";
import { IconButton } from "./IconButton";

interface FilterCardProps {
  filter: FilterGroup;
  associatedJobCount: number;
  onToggle: (enabled: boolean) => Promise<void>;
  onDelete: (deleteAssociatedJobs: boolean) => Promise<void>;
}

export function FilterCard({
  filter,
  associatedJobCount,
  onToggle,
  onDelete,
}: FilterCardProps) {
  const [confirming, setConfirming] = useState(false);
  const [deleteAssociatedJobs, setDeleteAssociatedJobs] = useState(false);
  const sourceCount = filter.FilterIDs.length;
  const isAllEnabled = filter.AllEnabled;
  const isMixedEnabled = filter.Enabled && !filter.AllEnabled;
  const sourcesSummary =
    filter.Sources.length === 1
      ? filter.Sources[0]
      : filter.Sources.join(", ");

  const handleDelete = () => {
    if (confirming) {
      void onDelete(deleteAssociatedJobs).catch(() => {
        // Parent tracks mutation errors for display.
      });
      setConfirming(false);
      setDeleteAssociatedJobs(false);
    } else {
      setConfirming(true);
      setDeleteAssociatedJobs(false);
    }
  };

  return (
    <div
      className={`p-3 rounded border transition-opacity ${
        filter.Enabled
          ? "border-hw-border bg-hw-surface"
          : "border-hw-border/50 bg-hw-surface/50 opacity-60"
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-sm font-semibold text-hw-text truncate leading-relaxed">
            {filter.Name}
          </p>
          <p className="text-xs text-hw-text-muted truncate mt-0.5 leading-relaxed">
            {filter.Keywords}
            {filter.Location && ` \u00B7 ${filter.Location}`}
          </p>
        </div>
        {confirming ? (
          <div className="flex gap-1 shrink-0">
            <Button
              variant="danger"
              size="sm"
              onClick={handleDelete}
              aria-label={`Confirm delete filter ${filter.Name}`}
            >
              Delete
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                setConfirming(false);
                setDeleteAssociatedJobs(false);
              }}
              aria-label="Cancel delete"
            >
              No
            </Button>
          </div>
        ) : (
          <IconButton
            aria-label={`Delete filter ${filter.Name}`}
            onClick={handleDelete}
            className="shrink-0 hover:text-hw-danger"
          >
            ✕
          </IconButton>
        )}
      </div>

      {confirming && associatedJobCount > 0 && (
        <label className="mt-2 flex items-center gap-2 text-xs text-hw-text-muted">
          <input
            type="checkbox"
            checked={deleteAssociatedJobs}
            onChange={(event) => setDeleteAssociatedJobs(event.target.checked)}
            className="h-3.5 w-3.5 rounded border-hw-border bg-hw-bg text-hw-accent focus:ring-hw-accent"
            aria-label={`Also delete ${associatedJobCount} associated jobs`}
          />
          Also delete {associatedJobCount} associated{" "}
          {associatedJobCount === 1 ? "job" : "jobs"}
        </label>
      )}

      <div className="flex items-center justify-between mt-2">
        <div className="min-w-0">
          <div className="text-xs text-hw-text-muted truncate">
            Sources: {sourcesSummary}
          </div>
          {sourceCount > 1 && (
            <div className="text-xs text-hw-text-muted">
              {filter.EnabledSourceCount}/{sourceCount} sources enabled
            </div>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            void onToggle(!isAllEnabled).catch(() => {
              // Parent tracks mutation errors for display.
            });
          }}
          className={`font-medium ${
            isAllEnabled
              ? "bg-hw-success/20 text-hw-success hover:bg-hw-success/30"
              : isMixedEnabled
                ? "bg-hw-accent/20 text-hw-accent hover:bg-hw-accent/30"
              : "bg-hw-text-muted/20 hover:bg-hw-text-muted/30"
          }`}
          aria-label={
            isAllEnabled
              ? `Disable filter ${filter.Name}`
              : `Enable filter ${filter.Name}`
          }
        >
          {isAllEnabled ? "ON" : isMixedEnabled ? "MIXED" : "OFF"}
        </Button>
      </div>
    </div>
  );
}
