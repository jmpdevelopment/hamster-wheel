import { useCallback, useEffect, useState } from "react";
import {
  GetFilters,
  CreateFilter,
  UpdateFilter,
  DeleteFilter,
} from "../../bindings/hamster-wheel/filterservice";
import { SearchFilter } from "../../bindings/hamster-wheel/internal/db/models";

export interface UseFiltersReturn {
  filters: SearchFilter[];
  loading: boolean;
  error: string | null;
  clearError: () => void;
  refresh: () => Promise<void>;
  createFilter: (
    name: string,
    keywords: string,
    location: string,
    source: string
  ) => Promise<string>;
  updateFilter: (
    id: string,
    name: string,
    keywords: string,
    location: string,
    source: string,
    enabled: boolean
  ) => Promise<void>;
  deleteFilter: (id: string) => Promise<void>;
}

export function useFilters(): UseFiltersReturn {
  const [filters, setFilters] = useState<SearchFilter[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const refresh = useCallback(async () => {
    try {
      setError(null);
      const fetched = await GetFilters();
      // Go nil slice → JS null, normalize to empty array.
      setFilters(fetched ?? []);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to fetch filters:", message);
      setError(message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const handleCreateFilter = useCallback(
    async (
      name: string,
      keywords: string,
      location: string,
      source: string
    ): Promise<string> => {
      try {
        setError(null);
        const id = await CreateFilter(name, keywords, location, source);
        await refresh();
        return id;
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to create filter:", message);
        setError(message);
        throw err;
      }
    },
    [refresh]
  );

  const handleUpdateFilter = useCallback(
    async (
      id: string,
      name: string,
      keywords: string,
      location: string,
      source: string,
      enabled: boolean
    ): Promise<void> => {
      try {
        setError(null);
        await UpdateFilter(id, name, keywords, location, source, enabled);
        await refresh();
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to update filter:", message);
        setError(message);
        throw err;
      }
    },
    [refresh]
  );

  const handleDeleteFilter = useCallback(
    async (id: string): Promise<void> => {
      try {
        setError(null);
        await DeleteFilter(id);
        await refresh();
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to delete filter:", message);
        setError(message);
        throw err;
      }
    },
    [refresh]
  );

  return {
    filters,
    loading,
    error,
    clearError,
    refresh,
    createFilter: handleCreateFilter,
    updateFilter: handleUpdateFilter,
    deleteFilter: handleDeleteFilter,
  };
}
