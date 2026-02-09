import { useState } from "react";
import { Job } from "../../bindings/hamster-wheel/internal/db/models";
import { formatDate, relativeTime } from "../lib/format";
import { containsHTML, sanitizeHTML } from "../lib/sanitize";
import { Browser } from "@wailsio/runtime";

interface JobDetailProps {
  job: Job;
  onDelete: (id: string) => void;
  onClose: () => void;
}

export function JobDetail({ job, onDelete, onClose }: JobDetailProps) {
  const [confirming, setConfirming] = useState(false);

  const handleDelete = () => {
    if (confirming) {
      onDelete(job.ID);
      setConfirming(false);
    } else {
      setConfirming(true);
    }
  };

  const handleOpenInBrowser = () => {
    if (job.URL) {
      Browser.OpenURL(job.URL);
    }
  };

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="shrink-0 px-4 py-3 border-b border-hw-border">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <h2 className="text-lg font-bold text-hw-text">{job.Title}</h2>
            <p className="text-sm text-hw-text-muted mt-0.5">
              {job.Company}
              {job.Company && job.Location && " \u00B7 "}
              {job.Location}
            </p>
          </div>
          <button
            onClick={onClose}
            className="shrink-0 text-hw-text-muted hover:text-hw-text text-lg"
            aria-label="Close detail"
          >
            ✕
          </button>
        </div>

        <div className="flex items-center gap-4 mt-2 text-xs text-hw-text-muted">
          <span>Posted: {formatDate(job.PostedAt)}</span>
          <span>Found: {relativeTime(job.DiscoveredAt)}</span>
          <span>{job.Source}</span>
        </div>

        <div className="flex gap-2 mt-3">
          {job.URL && (
            <button
              onClick={handleOpenInBrowser}
              className="px-3 py-1.5 text-xs font-medium rounded bg-hw-accent text-hw-bg hover:bg-hw-accent-hover transition-colors"
            >
              Open in Browser
            </button>
          )}
          {confirming ? (
            <>
              <button
                onClick={handleDelete}
                className="px-3 py-1.5 text-xs font-medium rounded bg-hw-danger text-white hover:bg-hw-danger/80 transition-colors"
                aria-label="Confirm delete"
              >
                Confirm Delete
              </button>
              <button
                onClick={() => setConfirming(false)}
                className="px-3 py-1.5 text-xs font-medium rounded border border-hw-border text-hw-text-muted hover:text-hw-text transition-colors"
                aria-label="Cancel delete"
              >
                Cancel
              </button>
            </>
          ) : (
            <button
              onClick={handleDelete}
              className="px-3 py-1.5 text-xs font-medium rounded border border-hw-danger/40 text-hw-danger hover:bg-hw-danger/10 transition-colors"
            >
              Delete
            </button>
          )}
        </div>
      </div>

      {/* Description */}
      <div className="flex-1 overflow-y-auto px-4 py-3">
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
          <p className="text-sm text-hw-text-muted italic">
            No description available.
          </p>
        )}
      </div>
    </div>
  );
}
