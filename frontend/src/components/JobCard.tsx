import { memo } from "react";
import { Job } from "../../bindings/hamster-wheel/internal/db/models";
import { Browser } from "@wailsio/runtime";
import { relativeTime } from "../lib/format";
import { sourceAttributionURL, sourceDisplayLabel } from "../lib/jobSource";
import {
  buildMatchStatusMeta,
  readMatchScore,
  readMatchStatus,
} from "../lib/matchStatus";

interface JobCardProps {
  job: Job;
  isSelected: boolean;
  isChecked: boolean;
  isFavorite: boolean;
  onClick: (shiftKey: boolean) => void;
  onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
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
  onContextMenu,
  onToggleChecked,
  onToggleFavorite,
  style,
}: JobCardProps) {
  const matchStatus = readMatchStatus(job);
  const matchScore = readMatchScore(job);
  const matchMeta = buildMatchStatusMeta(matchStatus, matchScore);
  const sourceLabel = sourceDisplayLabel(job.Source);
  const sourceLink = sourceAttributionURL(job.Source);
  const handleSourceLinkClick = (event: React.MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (sourceLink !== "") {
      Browser.OpenURL(sourceLink);
    }
  };

  return (
    <div
      style={style}
      className={`border-b border-hw-border transition-colors duration-150 ${
        isSelected
          ? "bg-hw-accent/10 border-l-2 border-l-hw-accent"
          : "hover:bg-hw-surface-hover"
      }`}
      onContextMenu={onContextMenu}
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

        <div className="flex-1 min-w-0">
          <button
            onClick={(event) => {
              onClick(event.shiftKey);
            }}
            aria-selected={isSelected}
            className="w-full text-left rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-hw-accent focus-visible:ring-offset-1 focus-visible:ring-offset-hw-bg"
          >
            <div className="flex items-start justify-between gap-2">
              <p className="text-sm font-semibold text-hw-text truncate leading-tight min-w-0">
                {job.Title}
              </p>
              {matchStatus !== "unknown" && (
                <span
                  aria-label={matchMeta.badgeAriaLabel}
                  className={`hw-match-badge hw-match-badge--compact shrink-0 ${matchMeta.badgeVariantClass}`}
                  title={matchMeta.badgeLabel}
                >
                  {matchMeta.badgeLabel}
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
          </button>
          <div className="mt-0.5 flex items-center justify-between gap-2">
            <span className="text-xs text-hw-text-muted shrink-0">
              {relativeTime(job.DiscoveredAt)}
            </span>
            {sourceLink !== "" ? (
              <a
                href={sourceLink}
                target="_blank"
                rel="noreferrer"
                onClick={handleSourceLinkClick}
                className="text-xs text-hw-text-muted/60 truncate max-w-[140px] underline underline-offset-2 hover:text-hw-text"
              >
                {sourceLabel}
              </a>
            ) : (
              <span className="text-xs text-hw-text-muted/60 truncate max-w-[140px]">
                {sourceLabel}
              </span>
            )}
          </div>
        </div>

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
