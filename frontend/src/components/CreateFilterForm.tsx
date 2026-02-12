import { useState } from "react";
import { Button } from "./Button";
import { Input } from "./Input";

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
      <div className="text-xs text-hw-text-muted">
        Source: {source}
      </div>
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
