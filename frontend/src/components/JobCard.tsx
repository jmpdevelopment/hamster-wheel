import { memo } from "react";
import { Job } from "../../bindings/hamster-wheel/internal/db/models";
import { relativeTime } from "../lib/format";

interface JobCardProps {
  job: Job;
  isSelected: boolean;
  onClick: () => void;
  style?: React.CSSProperties;
}

export const JobCard = memo(function JobCard({
  job,
  isSelected,
  onClick,
  style,
}: JobCardProps) {
  return (
    <div style={style}>
      <button
        onClick={onClick}
        aria-selected={isSelected}
        className={`w-full text-left px-3 py-2.5 border-b border-hw-border transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-hw-accent focus-visible:ring-offset-1 focus-visible:ring-offset-hw-bg ${
          isSelected
            ? "bg-hw-accent/10 border-l-2 border-l-hw-accent"
            : "hover:bg-hw-surface-hover"
        }`}
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
    </div>
  );
});
