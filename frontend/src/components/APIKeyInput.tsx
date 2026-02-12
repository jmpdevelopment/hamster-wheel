import { useState, useEffect, useRef } from "react";
import { GetReedAPIKey, SetReedAPIKey } from "../../bindings/hamster-wheel/settingsservice";
import { Button } from "./Button";
import { Input } from "./Input";

interface APIKeyInputProps {
  onError: (msg: string) => void;
}

export function APIKeyInput({ onError }: APIKeyInputProps) {
  const [key, setKey] = useState("");
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);
  const [hasKey, setHasKey] = useState(false);
  const savedTimeoutRef = useRef<number | null>(null);

  useEffect(() => {
    GetReedAPIKey()
      .then((k) => {
        if (k) {
          setKey(k);
          setHasKey(true);
        }
      })
      .catch((err: unknown) => {
        console.error("Failed to load API key:", err);
      });
  }, []);

  // Cleanup timeout on unmount.
  useEffect(() => {
    return () => {
      if (savedTimeoutRef.current !== null) {
        clearTimeout(savedTimeoutRef.current);
      }
    };
  }, []);

  const handleSave = async () => {
    const trimmed = key.trim();
    if (!trimmed) return;

    setSaving(true);
    setSaved(false);
    try {
      await SetReedAPIKey(trimmed);
      setHasKey(true);
      setSaved(true);

      // Clear any existing timeout before setting a new one.
      if (savedTimeoutRef.current !== null) {
        clearTimeout(savedTimeoutRef.current);
      }

      // Reset saved state after 2 seconds.
      savedTimeoutRef.current = window.setTimeout(() => {
        setSaved(false);
        savedTimeoutRef.current = null;
      }, 2000);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save API key:", message);
      onError(message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="px-3 py-2 border-b border-hw-border">
      <label className="block text-xs font-medium text-hw-text-muted mb-1">
        Reed API Key
      </label>
      <div className="flex gap-1">
        <Input
          type="password"
          size="sm"
          value={key}
          onChange={(e) => {
            setKey(e.target.value);
            setSaved(false);
          }}
          placeholder="Enter API key"
          className="flex-1 min-w-0"
          aria-label="Reed API Key"
        />
        <Button
          variant="primary"
          size="sm"
          onClick={handleSave}
          disabled={!key.trim()}
          loading={saving}
          className="shrink-0"
        >
          {saved ? "Saved" : "Save"}
        </Button>
      </div>
      {!hasKey && (
        <p className="text-xs text-hw-accent mt-1 leading-relaxed">
          Get a free key at reed.co.uk/developers
        </p>
      )}
    </div>
  );
}
