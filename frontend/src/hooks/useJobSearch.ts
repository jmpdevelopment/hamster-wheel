import { useState, useMemo, useEffect, useRef } from "react";
import { Job } from "../../bindings/hamster-wheel/internal/db/models";
import { readMatchScore, readMatchStatus } from "../lib/matchStatus";

interface SearchEntry {
  job: Job;
  searchText: string;
  postedAtMs: number | null;
  matchScore: number | null;
  hasScoredMatch: boolean;
  originalIndex: number;
}

export type JobSortMode =
  | "posted-desc"
  | "posted-asc"
  | "score-desc"
  | "score-asc";
export type PostedDateFilterMode =
  | "any"
  | "last-24h"
  | "last-7d"
  | "last-30d";
export type MatchScoreFilterMode =
  | "any"
  | "scored"
  | "score-80"
  | "score-60"
  | "score-40";

export interface UseJobSearchReturn {
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  sortMode: JobSortMode;
  setSortMode: (mode: JobSortMode) => void;
  postedDateFilterMode: PostedDateFilterMode;
  setPostedDateFilterMode: (mode: PostedDateFilterMode) => void;
  matchScoreFilterMode: MatchScoreFilterMode;
  setMatchScoreFilterMode: (mode: MatchScoreFilterMode) => void;
  filteredJobs: Job[];
}

function parseDateMs(date: unknown): number | null {
  if (typeof date === "string") {
    const ms = Date.parse(date);
    return Number.isNaN(ms) ? null : ms;
  }
  if (typeof date === "number") {
    return Number.isFinite(date) ? date : null;
  }
  return null;
}

function compareNullableNumber(
  left: number | null,
  right: number | null,
  mode: "asc" | "desc"
): number {
  if (left === null && right === null) return 0;
  if (left === null) return 1;
  if (right === null) return -1;
  return mode === "asc" ? left - right : right - left;
}

function compareBySortMode(
  left: SearchEntry,
  right: SearchEntry,
  sortMode: JobSortMode
): number {
  switch (sortMode) {
    case "posted-asc":
      return compareNullableNumber(left.postedAtMs, right.postedAtMs, "asc");
    case "score-desc":
      return compareNullableNumber(left.matchScore, right.matchScore, "desc");
    case "score-asc":
      return compareNullableNumber(left.matchScore, right.matchScore, "asc");
    case "posted-desc":
    default:
      return compareNullableNumber(left.postedAtMs, right.postedAtMs, "desc");
  }
}

function matchScoreThreshold(
  mode: MatchScoreFilterMode
): number | null {
  switch (mode) {
    case "score-80":
      return 0.8;
    case "score-60":
      return 0.6;
    case "score-40":
      return 0.4;
    default:
      return null;
  }
}

function buildSearchIndex(jobs: Job[]): SearchEntry[] {
  return jobs.map((job, index) => {
    const postedAtMs = parseDateMs(job.PostedAt);
    const matchScore = readMatchScore(job);
    const hasScoredMatch =
      readMatchStatus(job) === "matched" && matchScore !== null;
    return {
      job,
      searchText: [job.Title, job.Company, job.Location, job.Description]
        .join(" ")
        .toLowerCase(),
      postedAtMs,
      // Non-matched jobs are treated as having no score for sort/filter.
      matchScore: hasScoredMatch ? matchScore : null,
      hasScoredMatch,
      originalIndex: index,
    };
  });
}

export function useJobSearch(
  jobs: Job[],
  filterByFilterId: string | null
): UseJobSearchReturn {
  const [searchTerm, setSearchTerm] = useState("");
  const [debouncedTerm, setDebouncedTerm] = useState("");
  const [sortMode, setSortMode] = useState<JobSortMode>("posted-desc");
  const [postedDateFilterMode, setPostedDateFilterMode] =
    useState<PostedDateFilterMode>("any");
  const [matchScoreFilterMode, setMatchScoreFilterMode] =
    useState<MatchScoreFilterMode>("any");
  const timerRef = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    timerRef.current = setTimeout(() => {
      setDebouncedTerm(searchTerm);
    }, 200);
    return () => clearTimeout(timerRef.current);
  }, [searchTerm]);

  const searchIndex = useMemo(() => buildSearchIndex(jobs), [jobs]);

  const filteredJobs = useMemo(() => {
    let results = searchIndex;

    if (filterByFilterId) {
      results = results.filter(
        (entry) => entry.job.FilterID === filterByFilterId
      );
    }

    const trimmed = debouncedTerm.trim();
    if (trimmed) {
      const terms = trimmed.toLowerCase().split(/\s+/);
      results = results.filter((entry) =>
        terms.every((term) => entry.searchText.includes(term))
      );
    }

    if (postedDateFilterMode !== "any") {
      const now = Date.now();
      const windowMs =
        postedDateFilterMode === "last-24h"
          ? 24 * 60 * 60 * 1000
          : postedDateFilterMode === "last-7d"
            ? 7 * 24 * 60 * 60 * 1000
            : 30 * 24 * 60 * 60 * 1000;
      const cutoff = now - windowMs;
      results = results.filter(
        (entry) => entry.postedAtMs !== null && entry.postedAtMs >= cutoff
      );
    }

    if (matchScoreFilterMode !== "any") {
      const threshold = matchScoreThreshold(matchScoreFilterMode);
      if (matchScoreFilterMode === "scored") {
        results = results.filter((entry) => entry.hasScoredMatch);
      } else if (threshold !== null) {
        results = results.filter(
          (entry) =>
            entry.hasScoredMatch &&
            entry.matchScore !== null &&
            entry.matchScore >= threshold
        );
      }
    }

    return [...results]
      .sort((left, right) => {
        const bySort = compareBySortMode(left, right, sortMode);
        if (bySort !== 0) {
          return bySort;
        }
        return left.originalIndex - right.originalIndex;
      })
      .map((entry) => entry.job);
  }, [
    searchIndex,
    debouncedTerm,
    filterByFilterId,
    postedDateFilterMode,
    matchScoreFilterMode,
    sortMode,
  ]);

  return {
    searchTerm,
    setSearchTerm,
    sortMode,
    setSortMode,
    postedDateFilterMode,
    setPostedDateFilterMode,
    matchScoreFilterMode,
    setMatchScoreFilterMode,
    filteredJobs,
  };
}
