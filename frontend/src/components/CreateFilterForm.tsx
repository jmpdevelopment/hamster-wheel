import { useMemo, useState } from "react";
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
    sources: string[]
  ) => Promise<void>;
  onCancel: () => void;
}

export function CreateFilterForm({ onSubmit, onCancel }: CreateFilterFormProps) {
  const [name, setName] = useState("");
  const [keywords, setKeywords] = useState("");
  const [location, setLocation] = useState("");
  const [selectedSources, setSelectedSources] = useState<string[]>([
    REED_SOURCE,
  ]);
  const [submitting, setSubmitting] = useState(false);
  const sourceOptions = listSourceOptions();
  const sourceNotices = useMemo(
    () =>
      Array.from(
        new Set(
          selectedSources
            .map((source) => sourceDescriptionNotice(source))
            .filter((notice) => notice !== "")
        )
      ),
    [selectedSources]
  );

  const canSubmit =
    name.trim() !== "" &&
    keywords.trim() !== "" &&
    selectedSources.length > 0 &&
    !submitting;

  const handleToggleSource = (source: string, checked: boolean) => {
    setSelectedSources((previous) => {
      if (checked) {
        return previous.includes(source) ? previous : [...previous, source];
      }
      return previous.filter((selectedSource) => selectedSource !== source);
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;

    setSubmitting(true);
    try {
      await onSubmit(
        name.trim(),
        keywords.trim(),
        location.trim(),
        selectedSources
      );
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
      <fieldset className="space-y-1">
        <legend className="text-xs text-hw-text-muted">Sources</legend>
        <div className="space-y-1 rounded border border-hw-border bg-hw-bg px-2 py-2">
          {sourceOptions.map((option) => (
            <label
              key={option.value}
              className="flex items-center gap-2 text-sm text-hw-text"
            >
              <input
                type="checkbox"
                checked={selectedSources.includes(option.value)}
                onChange={(event) =>
                  handleToggleSource(option.value, event.target.checked)
                }
                className="h-4 w-4 rounded border-hw-border bg-hw-bg text-hw-accent focus:ring-hw-accent"
                aria-label={option.label}
              />
              <span>{option.label}</span>
            </label>
          ))}
        </div>
      </fieldset>
      <div className="text-xs text-hw-text-muted">
        Sources:{" "}
        {selectedSources.length > 0 ? selectedSources.join(", ") : "None"}
      </div>
      {sourceNotices.map((notice) => (
        <div key={notice} className="text-xs text-hw-text-muted">
          {notice}
        </div>
      ))}
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
