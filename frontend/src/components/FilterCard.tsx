import { useState } from "react";
import { SearchFilter } from "../../bindings/hamster-wheel/internal/db/models";

interface FilterCardProps {
  filter: SearchFilter;
  onToggle: (enabled: boolean) => void;
  onDelete: () => void;
}

export function FilterCard({ filter, onToggle, onDelete }: FilterCardProps) {
  const [confirming, setConfirming] = useState(false);

  const handleDelete = () => {
    if (confirming) {
      onDelete();
      setConfirming(false);
    } else {
      setConfirming(true);
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
          <p className="text-sm font-semibold text-hw-text truncate">
            {filter.Name}
          </p>
          <p className="text-xs text-hw-text-muted truncate mt-0.5">
            {filter.Keywords}
            {filter.Location && ` \u00B7 ${filter.Location}`}
          </p>
        </div>
        {confirming ? (
          <div className="flex gap-1 shrink-0">
            <button
              onClick={handleDelete}
              className="text-xs px-1.5 py-0.5 rounded bg-hw-danger text-white"
              aria-label={`Confirm delete filter ${filter.Name}`}
            >
              Delete
            </button>
            <button
              onClick={() => setConfirming(false)}
              className="text-xs px-1.5 py-0.5 rounded border border-hw-border text-hw-text-muted"
              aria-label="Cancel delete"
            >
              No
            </button>
          </div>
        ) : (
          <button
            onClick={handleDelete}
            className="shrink-0 text-hw-text-muted hover:text-hw-danger text-xs"
            aria-label={`Delete filter ${filter.Name}`}
          >
            ✕
          </button>
        )}
      </div>

      <div className="flex items-center justify-between mt-2">
        <span className="text-xs text-hw-text-muted">{filter.Source}</span>
        <button
          onClick={() => onToggle(!filter.Enabled)}
          className={`text-xs px-2 py-0.5 rounded font-medium transition-colors ${
            filter.Enabled
              ? "bg-hw-success/20 text-hw-success"
              : "bg-hw-text-muted/20 text-hw-text-muted"
          }`}
          aria-label={
            filter.Enabled
              ? `Disable filter ${filter.Name}`
              : `Enable filter ${filter.Name}`
          }
        >
          {filter.Enabled ? "ON" : "OFF"}
        </button>
      </div>
    </div>
  );
}
