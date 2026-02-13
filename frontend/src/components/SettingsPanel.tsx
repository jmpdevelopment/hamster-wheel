import { useState, useEffect, useRef } from "react";
import {
  GetReedAPIKey,
  SetReedAPIKey,
} from "../../bindings/hamster-wheel/settingsservice";
import { Button } from "./Button";
import { IconButton } from "./IconButton";
import { Input } from "./Input";
import type { ThemePreference } from "../hooks/useTheme";

interface SettingsPanelProps {
  onClose: () => void;
  theme: ThemePreference;
  onSetTheme: (theme: ThemePreference) => Promise<void>;
  onError: (msg: string) => void;
}

const themeOptions: { value: ThemePreference; label: string }[] = [
  { value: "system", label: "System" },
  { value: "dark", label: "Dark" },
  { value: "light", label: "Light" },
];

export function SettingsPanel({
  onClose,
  theme,
  onSetTheme,
  onError,
}: SettingsPanelProps) {
  // --- Reed API Key state (migrated from APIKeyInput) ---
  const [apiKey, setApiKey] = useState("");
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);
  const [hasKey, setHasKey] = useState(false);
  const savedTimeoutRef = useRef<number | null>(null);

  useEffect(() => {
    GetReedAPIKey()
      .then((k) => {
        if (k) {
          setApiKey(k);
          setHasKey(true);
        }
      })
      .catch((err: unknown) => {
        console.error("Failed to load API key:", err);
      });
  }, []);

  useEffect(() => {
    return () => {
      if (savedTimeoutRef.current !== null) {
        clearTimeout(savedTimeoutRef.current);
      }
    };
  }, []);

  const handleSaveKey = async () => {
    const trimmed = apiKey.trim();
    if (!trimmed) return;

    setSaving(true);
    setSaved(false);
    try {
      await SetReedAPIKey(trimmed);
      setHasKey(true);
      setSaved(true);

      if (savedTimeoutRef.current !== null) {
        clearTimeout(savedTimeoutRef.current);
      }
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
    <div
      className="absolute top-0 right-0 bottom-0 w-[400px] bg-hw-surface border-l border-hw-border shadow-lg z-10 flex flex-col overflow-hidden transition-opacity duration-200"
      role="dialog"
      aria-label="Settings"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-hw-border shrink-0">
        <h2 className="text-lg font-bold text-hw-text">Settings</h2>
        <IconButton aria-label="Close settings" onClick={onClose}>
          ✕
        </IconButton>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        {/* Reed API Key Section */}
        <section>
          <h3 className="text-sm font-semibold text-hw-text mb-2">
            Reed API Key
          </h3>
          <div className="flex gap-2">
            <Input
              type="password"
              size="sm"
              value={apiKey}
              onChange={(e) => {
                setApiKey(e.target.value);
                setSaved(false);
              }}
              placeholder="Enter API key"
              className="flex-1 min-w-0"
              aria-label="Reed API Key"
            />
            <Button
              variant="primary"
              size="sm"
              onClick={handleSaveKey}
              disabled={!apiKey.trim()}
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
        </section>

        {/* Theme Section */}
        <section>
          <h3 className="text-sm font-semibold text-hw-text mb-2">Theme</h3>
          <div className="flex gap-2">
            {themeOptions.map((opt) => (
              <Button
                key={opt.value}
                variant={theme === opt.value ? "primary" : "secondary"}
                size="sm"
                onClick={() => onSetTheme(opt.value)}
                aria-pressed={theme === opt.value}
              >
                {opt.label}
              </Button>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}
