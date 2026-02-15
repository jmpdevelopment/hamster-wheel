import { useState } from "react";
import { Button } from "./Button";
import { Input } from "./Input";
import {
  REED_SOURCE,
  listSourceOptions,
  sourceDescriptionNotice,
} from "../lib/jobSource";

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
  const [source, setSource] = useState(REED_SOURCE);
  const [submitting, setSubmitting] = useState(false);
  const sourceNotice = sourceDescriptionNotice(source);

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
      <Input
        type="text"
        size="md"
        placeholder="Filter name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        aria-label="Filter name"
      />
      <Input
        type="text"
        size="md"
        placeholder="Keywords (e.g., go developer)"
        value={keywords}
        onChange={(e) => setKeywords(e.target.value)}
        aria-label="Keywords"
      />
      <Input
        type="text"
        size="md"
        placeholder="Location (optional)"
        value={location}
        onChange={(e) => setLocation(e.target.value)}
        aria-label="Location"
      />
      <label className="block text-xs text-hw-text-muted">
        Source
        <select
          value={source}
          onChange={(e) => setSource(e.target.value)}
          className="mt-1 w-full rounded bg-hw-bg border border-hw-border text-hw-text text-sm px-2 py-1.5 focus:outline-none focus:border-hw-accent"
          aria-label="Source"
        >
          {listSourceOptions().map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </label>
      <div className="text-xs text-hw-text-muted">
        Source: {source}
      </div>
      {sourceNotice && (
        <div className="text-xs text-hw-text-muted">
          {sourceNotice}
        </div>
      )}
      <div className="flex gap-2">
        <Button
          variant="primary"
          size="md"
          type="submit"
          disabled={!canSubmit}
          loading={submitting}
          className="flex-1"
        >
          Create
        </Button>
        <Button
          variant="secondary"
          size="md"
          type="button"
          onClick={onCancel}
        >
          Cancel
        </Button>
      </div>
    </form>
  );
}
