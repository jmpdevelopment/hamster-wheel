export type MatchStatus =
  | "pending"
  | "processing"
  | "matched"
  | "failed"
  | "unknown";

export interface MatchStatusMeta {
  badgeAriaLabel: string;
  badgeLabel: string;
  badgeVariantClass: string;
  detailHeadline: string;
  detailStatusLabel: string;
  status: MatchStatus;
}

interface MatchReadable {
  MatchScore?: unknown;
  MatchStatus?: unknown;
  MatchSummary?: unknown;
}

const providerPrefix = "provider:";

function parseMatchSummary(value: unknown): { provider: string; summary: string } {
  if (typeof value !== "string") {
    return { provider: "", summary: "" };
  }

  const trimmed = value.trim();
  if (!trimmed) {
    return { provider: "", summary: "" };
  }

  const lines = trimmed.split(/\r?\n/);
  const firstLine = lines[0].trim();
  if (!firstLine.toLowerCase().startsWith(providerPrefix)) {
    return { provider: "", summary: trimmed };
  }

  const provider = firstLine.slice(providerPrefix.length).trim();
  const summary = lines.slice(1).join("\n").trim();
  return { provider, summary };
}

export function readMatchStatus(job: MatchReadable): MatchStatus {
  const candidate = job.MatchStatus;
  if (typeof candidate !== "string") {
    return "unknown";
  }
  switch (candidate.trim().toLowerCase()) {
    case "pending":
      return "pending";
    case "processing":
      return "processing";
    case "matched":
      return "matched";
    case "failed":
      return "failed";
    default:
      return "unknown";
  }
}

export function readMatchScore(job: MatchReadable): number | null {
  const candidate = job.MatchScore;
  if (typeof candidate !== "number") {
    return null;
  }
  if (!Number.isFinite(candidate) || candidate < 0 || candidate > 1) {
    return null;
  }
  return candidate;
}

export function readMatchSummary(job: MatchReadable): string {
  return parseMatchSummary(job.MatchSummary).summary;
}

export function readMatchProvider(job: MatchReadable): string {
  return parseMatchSummary(job.MatchSummary).provider;
}

export function buildMatchStatusMeta(
  status: MatchStatus,
  score: number | null
): MatchStatusMeta {
  switch (status) {
    case "pending":
      return {
        status,
        badgeAriaLabel: "Match status: pending",
        badgeLabel: "Match pending",
        badgeVariantClass: "hw-match-badge--pending",
        detailHeadline: "Match queued for calculation",
        detailStatusLabel: "Queued",
      };
    case "processing":
      return {
        status,
        badgeAriaLabel: "Match status: processing",
        badgeLabel: "Matching",
        badgeVariantClass: "hw-match-badge--processing",
        detailHeadline: "Calculating match score...",
        detailStatusLabel: "Calculating",
      };
    case "matched": {
      const hasScore = score !== null;
      const percent = hasScore ? Math.round(score * 100) : null;
      return {
        status,
        badgeAriaLabel: "Match status: matched",
        badgeLabel: hasScore
          ? `Match Score: ${percent}%`
          : "Match score unavailable",
        badgeVariantClass: "hw-match-badge--matched",
        detailHeadline: hasScore
          ? `Match Score: ${percent}%`
          : "Match score unavailable",
        detailStatusLabel: "Matched",
      };
    }
    case "failed":
      return {
        status,
        badgeAriaLabel: "Match status: failed",
        badgeLabel: "Match failed",
        badgeVariantClass: "hw-match-badge--failed",
        detailHeadline: "Match calculation failed",
        detailStatusLabel: "Failed",
      };
    default:
      return {
        status,
        badgeAriaLabel: "Match status: not scored",
        badgeLabel: "Not scored",
        badgeVariantClass: "hw-match-badge--neutral",
        detailHeadline: "Match not calculated yet",
        detailStatusLabel: "Not scored",
      };
  }
}
