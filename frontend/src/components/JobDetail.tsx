import { useState } from "react";
import { Job } from "../../bindings/hamster-wheel/internal/db/models";
import {
  RecalculateMatchScore,
  RetryFetchDescription,
} from "../../bindings/hamster-wheel/jobservice";
import { formatDate, relativeTime } from "../lib/format";
import {
  buildMatchStatusMeta,
  readMatchProvider,
  readMatchScore,
  readMatchStatus,
  readMatchSummary,
} from "../lib/matchStatus";
import { sourceDescriptionNotice, sourceDisplayLabel } from "../lib/jobSource";
import { containsHTML, sanitizeHTML } from "../lib/sanitize";
import { toSafeExternalURL } from "../lib/url";
import { Browser } from "@wailsio/runtime";
import { Button } from "./Button";
import { IconButton } from "./IconButton";

interface JobDetailProps {
  job: Job;
  onDelete: (id: string) => Promise<void>;
  onClose: () => void;
  onRefresh: () => Promise<void>;
}

export function JobDetail({ job, onDelete, onClose, onRefresh }: JobDetailProps) {
  const [confirming, setConfirming] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [retryError, setRetryError] = useState<string | null>(null);
  const [recalculating, setRecalculating] = useState(false);
  const [recalculateError, setRecalculateError] = useState<string | null>(null);
  const [recalculateQueued, setRecalculateQueued] = useState(false);

  const matchStatus = readMatchStatus(job);
  const matchScore = readMatchScore(job);
  const matchSummary = readMatchSummary(job);
  const matchProvider = readMatchProvider(job);
  const matchMeta = buildMatchStatusMeta(matchStatus, matchScore);

  const matchProviderLabel =
    matchProvider === "openai"
      ? "OpenAI"
      : matchProvider;
  const sourceNotice = sourceDescriptionNotice(job.Source);
  const safeJobURL = toSafeExternalURL(job.URL);

  const handleDelete = () => {
    if (confirming) {
      void onDelete(job.ID).catch(() => {
        // Parent tracks mutation errors for display.
      });
      setConfirming(false);
    } else {
      setConfirming(true);
    }
  };

  const handleOpenInBrowser = () => {
    if (safeJobURL) {
      Browser.OpenURL(safeJobURL);
    }
  };

  const handleRetryDescription = async () => {
    setRetrying(true);
    setRetryError(null);
    try {
      await RetryFetchDescription(job.ID);
      await onRefresh();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      setRetryError(message);
    } finally {
      setRetrying(false);
    }
  };

  const handleRecalculateScore = async () => {
    setRecalculating(true);
    setRecalculateError(null);
    setRecalculateQueued(false);
    try {
      await RecalculateMatchScore(job.ID);
      await onRefresh();
      setRecalculateQueued(true);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      setRecalculateError(message);
    } finally {
      setRecalculating(false);
    }
  };

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="shrink-0 px-4 py-3 border-b border-hw-border">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <h2 className="text-lg font-bold text-hw-text leading-relaxed">{job.Title}</h2>
            <p className="text-sm text-hw-text-muted mt-0.5 leading-relaxed">
              {job.Company}
              {job.Company && job.Location && " \u00B7 "}
              {job.Location}
            </p>
          </div>
          <IconButton
            aria-label="Close detail"
            onClick={onClose}
            className="shrink-0"
          >
            ✕
          </IconButton>
        </div>

        <div className="flex items-center gap-4 mt-2 text-xs text-hw-text-muted">
          <span>Posted: {formatDate(job.PostedAt)}</span>
          <span>Found: {relativeTime(job.DiscoveredAt)}</span>
          <span>{sourceDisplayLabel(job.Source)}</span>
        </div>

        <div className="mt-3 rounded-md border border-hw-border bg-hw-surface/40 px-3 py-2">
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs font-semibold text-hw-text">
              {matchMeta.detailHeadline}
            </span>
            <span
              className={`hw-match-badge hw-match-badge--regular ${matchMeta.badgeVariantClass}`}
            >
              {matchMeta.detailStatusLabel}
            </span>
          </div>
          {matchProviderLabel && (
            <p className="mt-1 text-xs text-hw-text-muted leading-relaxed">
              LLM provider: {matchProviderLabel}
            </p>
          )}
          {matchSummary && (
            <p className="mt-1 text-xs text-hw-text-muted leading-relaxed">
              {matchSummary}
            </p>
          )}
          {recalculateQueued && (
            <p className="mt-1 text-xs text-hw-success">
              Recalculation queued.
            </p>
          )}
          {recalculateError && (
            <p className="mt-1 text-xs text-hw-danger">
              {recalculateError}
            </p>
          )}
        </div>

        <div className="flex gap-2 mt-3">
          {safeJobURL && (
            <Button
              variant="primary"
              size="sm"
              onClick={handleOpenInBrowser}
            >
              Open in Browser
            </Button>
          )}
          <Button
            variant="secondary"
            size="sm"
            onClick={handleRecalculateScore}
            disabled={recalculating || matchStatus === "processing"}
            loading={recalculating}
          >
            {recalculating
              ? "Recalculating..."
              : matchStatus === "processing"
                ? "Calculating..."
                : "Recalculate score"}
          </Button>
          {confirming ? (
            <>
              <Button
                variant="danger"
                size="sm"
                onClick={handleDelete}
                aria-label="Confirm delete"
              >
                Confirm Delete
              </Button>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setConfirming(false)}
                aria-label="Cancel delete"
              >
                Cancel
              </Button>
            </>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleDelete}
              className="border border-hw-danger/40 text-hw-danger hover:bg-hw-danger/10"
              aria-label="Delete job"
            >
              Delete
            </Button>
          )}
        </div>
      </div>

      {/* Description */}
      <div className="flex-1 overflow-y-auto px-4 py-3">
        {sourceNotice && (
          <p className="mb-3 rounded border border-hw-border bg-hw-surface/50 px-3 py-2 text-xs text-hw-text-muted leading-relaxed">
            {sourceNotice}
          </p>
        )}
        {job.Description ? (
          containsHTML(job.Description) ? (
            <div
              className="text-sm text-hw-text leading-relaxed space-y-2 [&_ul]:list-disc [&_ul]:ml-4 [&_ol]:list-decimal [&_ol]:ml-4 [&_li]:mb-1 [&_a]:text-hw-accent [&_a]:underline"
              dangerouslySetInnerHTML={{
                __html: sanitizeHTML(job.Description),
              }}
            />
          ) : (
            <pre className="text-sm text-hw-text whitespace-pre-wrap font-sans leading-relaxed">
              {job.Description}
            </pre>
          )
        ) : (
          <div className="space-y-3">
            <p className="text-sm text-hw-text-muted italic leading-relaxed">
              Description couldn't be loaded.
            </p>
            <Button
              variant="primary"
              size="sm"
              onClick={handleRetryDescription}
              disabled={retrying}
              loading={retrying}
            >
              {retrying ? "Retrying..." : "Retry"}
            </Button>
            {retryError && (
              <p className="text-xs text-hw-danger flex items-center gap-1 leading-relaxed">
                <svg
                  aria-hidden="true"
                  className="shrink-0 w-3 h-3"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
                  <line x1="12" y1="9" x2="12" y2="13" />
                  <line x1="12" y1="17" x2="12.01" y2="17" />
                </svg>
                {retryError}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
