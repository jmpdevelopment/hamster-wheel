import { memo } from "react";
import { Job } from "../../bindings/hamster-wheel/internal/db/models";
import { relativeTime } from "../lib/format";

interface JobCardProps {
  job: Job;
  isSelected: boolean;
  isChecked: boolean;
  isFavorite: boolean;
  onClick: (shiftKey: boolean) => void;
  onToggleChecked: (checked: boolean, shiftKey: boolean) => void;
  onToggleFavorite: () => void;
  style?: React.CSSProperties;
}

export const JobCard = memo(function JobCard({
  job,
  isSelected,
  isChecked,
  isFavorite,
  onClick,
  onToggleChecked,
  onToggleFavorite,
  style,
}: JobCardProps) {
  return (
    <div
      style={style}
      className={`border-b border-hw-border transition-colors duration-150 ${
        isSelected
          ? "bg-hw-accent/10 border-l-2 border-l-hw-accent"
          : "hover:bg-hw-surface-hover"
      }`}
    >
      <div className="flex items-start gap-2 px-3 py-2.5">
        <input
          type="checkbox"
          checked={isChecked}
          readOnly
          onClick={(event) => {
            onToggleChecked(!isChecked, event.shiftKey);
          }}
          className="mt-1.5 h-4 w-4 rounded border-hw-border bg-hw-bg text-hw-accent focus:ring-hw-accent"
          aria-label={`Select job ${job.Title}`}
        />

        <button
          onClick={(event) => {
            onClick(event.shiftKey);
          }}
          aria-selected={isSelected}
          className="flex-1 min-w-0 text-left rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-hw-accent focus-visible:ring-offset-1 focus-visible:ring-offset-hw-bg"
        >
          <p className="text-sm font-semibold text-hw-text truncate leading-relaxed">
            {job.Title}
          </p>
          <div className="flex items-center gap-2 mt-0.5">
            {job.Company && (
              <span className="text-xs text-hw-text-muted truncate">
                {job.Company}
              </span>
            )}
            {job.Company && job.Location && (
              <span className="text-xs text-hw-text-muted">&middot;</span>
            )}
            {job.Location && (
              <span className="text-xs text-hw-text-muted truncate">
                {job.Location}
              </span>
            )}
          </div>
          <div className="flex items-center justify-between mt-1">
            <span className="text-xs text-hw-text-muted">
              {relativeTime(job.DiscoveredAt)}
            </span>
            <span className="text-xs text-hw-text-muted/60">{job.Source}</span>
          </div>
        </button>

        <button
          onClick={onToggleFavorite}
          aria-label={
            isFavorite
              ? `Remove ${job.Title} from favorites`
              : `Add ${job.Title} to favorites`
          }
          className={`mt-0.5 rounded px-1.5 py-0.5 text-sm transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-hw-accent focus-visible:ring-offset-1 focus-visible:ring-offset-hw-bg ${
            isFavorite
              ? "text-amber-400 hover:text-amber-300"
              : "text-hw-text-muted hover:text-hw-text"
          }`}
        >
          {isFavorite ? "★" : "☆"}
        </button>
      </div>
    </div>
  );
});
