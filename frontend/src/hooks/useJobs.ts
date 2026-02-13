import { useCallback, useEffect, useState } from "react";
import {
  GetJobs,
  GetJobCount,
  DeleteJob,
} from "../../bindings/hamster-wheel/jobservice";
import { Job } from "../../bindings/hamster-wheel/internal/db/models";

export interface UseJobsReturn {
  jobs: Job[];
  jobCount: number;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  deleteJob: (id: string) => Promise<void>;
}

export function useJobs(limit: number = 0): UseJobsReturn {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [jobCount, setJobCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
        await DeleteJob(id);
        await refresh();
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to delete job:", message);
        setError(message);
      }
    },
    [refresh]
  );

  return {
    jobs,
    jobCount,
    loading,
    error,
    refresh,
    deleteJob: handleDeleteJob,
  };
}
