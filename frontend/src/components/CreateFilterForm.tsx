import { useState } from "react";

interface CreateFilterFormProps {
  onSubmit: (
    name: string,
    keywords: string,
    location: string,
    source: string
  ) => Promise<void>;
  onCancel: () => void;
}

export function CreateFilterForm({ onSubmit, onCancel }: CreateFilterFormProps) {
  const [name, setName] = useState("");
  const [keywords, setKeywords] = useState("");
  const [location, setLocation] = useState("");
  const [source] = useState("reed_uk");
  const [submitting, setSubmitting] = useState(false);

  const canSubmit = name.trim() !== "" && keywords.trim() !== "" && !submitting;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;

    setSubmitting(true);
    try {
      await onSubmit(name.trim(), keywords.trim(), location.trim(), source);
      // Reset form on success.
      setName("");
      setKeywords("");
      setLocation("");
    } catch {
      // Error is handled by the parent via the hook's error state.
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="p-3 rounded border border-hw-accent/30 bg-hw-surface space-y-2">
      <input
        type="text"
        placeholder="Filter name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        className="w-full px-2 py-1.5 text-sm rounded bg-hw-bg border border-hw-border text-hw-text placeholder-hw-text-muted focus:outline-none focus:border-hw-accent"
        aria-label="Filter name"
      />
      <input
        type="text"
        placeholder="Keywords (e.g., go developer)"
        value={keywords}
        onChange={(e) => setKeywords(e.target.value)}
        className="w-full px-2 py-1.5 text-sm rounded bg-hw-bg border border-hw-border text-hw-text placeholder-hw-text-muted focus:outline-none focus:border-hw-accent"
        aria-label="Keywords"
      />
      <input
        type="text"
        placeholder="Location (optional)"
        value={location}
        onChange={(e) => setLocation(e.target.value)}
        className="w-full px-2 py-1.5 text-sm rounded bg-hw-bg border border-hw-border text-hw-text placeholder-hw-text-muted focus:outline-none focus:border-hw-accent"
        aria-label="Location"
      />
      <div className="text-xs text-hw-text-muted">
        Source: {source}
      </div>
      <div className="flex gap-2">
        <button
          type="submit"
          disabled={!canSubmit}
          className="flex-1 px-2 py-1.5 text-sm font-medium rounded bg-hw-accent text-hw-bg hover:bg-hw-accent-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {submitting ? "Creating..." : "Create"}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-2 py-1.5 text-sm rounded border border-hw-border text-hw-text-muted hover:text-hw-text hover:border-hw-text-muted transition-colors"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
