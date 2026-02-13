import { useState, useMemo, useEffect, useRef } from "react";
import { Job } from "../../bindings/hamster-wheel/internal/db/models";

interface SearchEntry {
  job: Job;
  searchText: string;
}

export interface UseJobSearchReturn {
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  filteredJobs: Job[];
}

function buildSearchIndex(jobs: Job[]): SearchEntry[] {
  return jobs.map((job) => ({
    job,
    searchText: [job.Title, job.Company, job.Location, job.Description]
      .join(" ")
      .toLowerCase(),
  }));
}

export function useJobSearch(
  jobs: Job[],
  filterByFilterId: string | null
): UseJobSearchReturn {
  const [searchTerm, setSearchTerm] = useState("");
  const [debouncedTerm, setDebouncedTerm] = useState("");
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

    return results.map((entry) => entry.job);
  }, [searchIndex, debouncedTerm, filterByFilterId]);

  return { searchTerm, setSearchTerm, filteredJobs };
}
