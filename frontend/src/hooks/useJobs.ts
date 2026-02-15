import { useCallback, useEffect, useState } from "react";
import {
  GetJobs,
  GetJobCount,
  DeleteJob,
  SetJobFavorite,
  SetJobsFavorite,
  RecalculateMatchScore,
} from "../../bindings/hamster-wheel/jobservice";
import { Job } from "../../bindings/hamster-wheel/internal/db/models";

const sqliteBusyPattern = /(database is locked|SQLITE_BUSY)/i;
const maxBusyRetryAttempts = 6;

function wait(ms: number): Promise<void> {
  return new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

async function deleteJobWithBusyRetry(id: string): Promise<void> {
  await callWithBusyRetry(() => DeleteJob(id));
}

async function callWithBusyRetry<T>(operation: () => Promise<T>): Promise<T> {
  let attempt = 0;
  for (;;) {
    try {
      return await operation();
    } catch (err: unknown) {
      attempt += 1;
      const message = err instanceof Error ? err.message : String(err);
      const canRetry =
        sqliteBusyPattern.test(message) && attempt < maxBusyRetryAttempts;
      if (!canRetry) {
        throw err;
      }
      // Short bounded backoff for transient SQLite writer lock contention.
      await wait(50 * attempt);
    }
  }
}

export interface UseJobsReturn {
  jobs: Job[];
  jobCount: number;
  loading: boolean;
  error: string | null;
  clearError: () => void;
  refresh: () => Promise<void>;
  deleteJob: (id: string) => Promise<void>;
  deleteJobs: (ids: string[]) => Promise<void>;
  setJobFavorite: (id: string, favorite: boolean) => Promise<void>;
  setJobsFavorite: (ids: string[], favorite: boolean) => Promise<void>;
  recalculateMatchScores: (ids: string[]) => Promise<void>;
}

export function useJobs(limit: number = 0): UseJobsReturn {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [jobCount, setJobCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const refresh = useCallback(async () => {
    try {
      setError(null);
      const [fetchedJobs, count] = await Promise.all([
        GetJobs(limit),
        GetJobCount(),
      ]);
      // Go nil slice → JS null, normalize to empty array.
      setJobs(fetchedJobs ?? []);
      setJobCount(count ?? 0);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to fetch jobs:", message);
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [limit]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleDeleteJob = useCallback(
    async (id: string) => {
      try {
        setError(null);
        await deleteJobWithBusyRetry(id);
        await refresh();
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to delete job:", message);
        setError(message);
        throw err;
      }
    },
    [refresh]
  );

  const handleDeleteJobs = useCallback(
    async (ids: string[]) => {
      const uniqueIDs = Array.from(new Set(ids)).filter((id) => id.length > 0);
      if (uniqueIDs.length === 0) {
        return;
      }

      try {
        setError(null);
        const failures: unknown[] = [];
        for (const id of uniqueIDs) {
          try {
            // Serialize deletes to avoid SQLite write-lock contention.
            await deleteJobWithBusyRetry(id);
          } catch (err: unknown) {
            failures.push(err);
          }
        }
        await refresh();
        if (failures.length > 0) {
          const firstReason = failures[0];
          const reason =
            firstReason instanceof Error
              ? firstReason.message
              : String(firstReason);
          const message = `Failed to delete ${failures.length} of ${uniqueIDs.length} jobs: ${reason}`;
          setError(message);
          throw new Error(message);
        }
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to delete jobs:", message);
        setError(message);
        throw err;
      }
    },
    [refresh]
  );

  const handleSetJobFavorite = useCallback(
    async (id: string, favorite: boolean) => {
      try {
        setError(null);
        await callWithBusyRetry(() => SetJobFavorite(id, favorite));
        await refresh();
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to update job favorite:", message);
        setError(message);
        throw err;
      }
    },
    [refresh]
  );

  const handleSetJobsFavorite = useCallback(
    async (ids: string[], favorite: boolean) => {
      const uniqueIDs = Array.from(new Set(ids)).filter((id) => id.length > 0);
      if (uniqueIDs.length === 0) {
        return;
      }

      try {
        setError(null);
        await callWithBusyRetry(() => SetJobsFavorite(uniqueIDs, favorite));
        await refresh();
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to update jobs favorite:", message);
        setError(message);
        throw err;
      }
    },
    [refresh]
  );

  const handleRecalculateMatchScores = useCallback(
    async (ids: string[]) => {
      const uniqueIDs = Array.from(new Set(ids)).filter((id) => id.length > 0);
      if (uniqueIDs.length === 0) {
        return;
      }

      try {
        setError(null);
        const failures: unknown[] = [];
        for (const id of uniqueIDs) {
          try {
            await callWithBusyRetry(() => RecalculateMatchScore(id));
          } catch (err: unknown) {
            failures.push(err);
          }
        }
        await refresh();
        if (failures.length > 0) {
          const firstReason = failures[0];
          const reason =
            firstReason instanceof Error
              ? firstReason.message
              : String(firstReason);
          const message = `Failed to queue recalculation for ${failures.length} of ${uniqueIDs.length} jobs: ${reason}`;
          setError(message);
          throw new Error(message);
        }
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to recalculate jobs:", message);
        setError(message);
        throw err;
      }
    },
    [refresh]
  );

  return {
    jobs,
    jobCount,
    loading,
    error,
    clearError,
    refresh,
    deleteJob: handleDeleteJob,
    deleteJobs: handleDeleteJobs,
    setJobFavorite: handleSetJobFavorite,
    setJobsFavorite: handleSetJobsFavorite,
    recalculateMatchScores: handleRecalculateMatchScores,
  };
}
