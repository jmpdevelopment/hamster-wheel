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
  const matchStatus = readMatchStatus(job);
  const matchStatusBadge = getMatchStatusBadge(matchStatus);

  return (
    <div
      style={style}
      className={`border-b border-hw-border transition-colors duration-150 ${
        isSelected
          ? "bg-hw-accent/10 border-l-2 border-l-hw-accent"
          : "hover:bg-hw-surface-hover"
      }`}
    >
      <div className="flex items-start gap-2 px-3 py-2">
        <input
          type="checkbox"
          checked={isChecked}
          readOnly
          onClick={(event) => {
            onToggleChecked(!isChecked, event.shiftKey);
          }}
          className="mt-1 h-4 w-4 rounded border-hw-border bg-hw-bg text-hw-accent focus:ring-hw-accent"
          aria-label={`Select job ${job.Title}`}
        />

        <button
          onClick={(event) => {
            onClick(event.shiftKey);
          }}
          aria-selected={isSelected}
          className="flex-1 min-w-0 text-left rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-hw-accent focus-visible:ring-offset-1 focus-visible:ring-offset-hw-bg"
        >
          <div className="flex items-start justify-between gap-2">
            <p className="text-sm font-semibold text-hw-text truncate leading-tight min-w-0">
              {job.Title}
            </p>
            {matchStatusBadge && (
              <span
                aria-label={matchStatusBadge.ariaLabel}
                className={`shrink-0 inline-flex items-center rounded-md border px-1.5 py-0 text-[10px] font-semibold leading-3 ${matchStatusBadge.className}`}
                title={matchStatusBadge.label}
              >
                {matchStatusBadge.label}
              </span>
            )}
          </div>
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
          <div className="flex items-center justify-between mt-0.5">
            <span className="text-xs text-hw-text-muted shrink-0">
              {relativeTime(job.DiscoveredAt)}
            </span>
            <span className="text-xs text-hw-text-muted/60 truncate max-w-[120px]">
              {job.Source}
            </span>
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

function readMatchStatus(job: Job): string {
  const candidate = (job as unknown as { MatchStatus?: unknown }).MatchStatus;
  if (typeof candidate !== "string") {
    return "";
  }
  return candidate.trim().toLowerCase();
}

type MatchStatusBadge = {
  ariaLabel: string;
  className: string;
  label: string;
};

function getMatchStatusBadge(status: string): MatchStatusBadge | null {
  switch (status) {
    case "pending":
      return {
        ariaLabel: "Match status: pending",
        className: "border-hw-accent/45 bg-hw-accent/10 text-hw-accent border-dashed",
        label: "Match pending",
      };
    case "processing":
      return {
        ariaLabel: "Match status: processing",
        className: "border-hw-accent/60 bg-hw-accent/15 text-hw-accent",
        label: "Matching",
      };
    case "matched":
      return {
        ariaLabel: "Match status: matched",
        className: "border-hw-success/45 bg-hw-success/10 text-hw-success",
        label: "Match ready",
      };
    case "failed":
      return {
        ariaLabel: "Match status: failed",
        className: "border-hw-danger/45 bg-hw-danger/10 text-hw-danger",
        label: "Match failed",
      };
    default:
      return null;
  }
}
