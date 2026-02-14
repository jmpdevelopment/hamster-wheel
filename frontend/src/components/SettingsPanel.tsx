import { useState, useEffect, useRef } from "react";
import { ShortcutsHelp } from "./ShortcutsHelp";
import {
  HasReedAPIKey,
  SetReedAPIKey,
  ClearReedAPIKey,
  SetKeyboardShortcuts,
} from "../../bindings/hamster-wheel/settingsservice";
import { Browser } from "@wailsio/runtime";
import { Button } from "./Button";
import { IconButton } from "./IconButton";
import { Input } from "./Input";
import type { ThemePreference } from "../hooks/useTheme";

interface SettingsPanelProps {
  onClose: () => void;
  theme: ThemePreference;
  onSetTheme: (theme: ThemePreference) => Promise<void>;
  onError: (msg: string) => void;
  keyboardShortcuts: boolean;
  onSetKeyboardShortcuts: (enabled: boolean) => void;
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
  keyboardShortcuts,
  onSetKeyboardShortcuts,
}: SettingsPanelProps) {
  // --- Reed API Key state (migrated from APIKeyInput) ---
  const [apiKey, setApiKey] = useState("");
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);
  const [clearing, setClearing] = useState(false);
  const [hasKey, setHasKey] = useState(false);
  const [shortcutsHelpOpen, setShortcutsHelpOpen] = useState(false);
  const savedTimeoutRef = useRef<number | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadHasKey = async () => {
      try {
        const present = await HasReedAPIKey();
        if (!cancelled) {
          setHasKey(Boolean(present));
        }
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to load API key:", message);
        if (!cancelled) {
          onError(message);
        }
      }
    };

    void loadHasKey();
    return () => {
      cancelled = true;
    };
  }, [onError]);

  useEffect(() => {
    return () => {
      if (savedTimeoutRef.current !== null) {
        clearTimeout(savedTimeoutRef.current);
      }
    };
  }, []);

  const handleSetKeyboardShortcuts = async (enabled: boolean) => {
    try {
      await SetKeyboardShortcuts(enabled ? "true" : "false");
      onSetKeyboardShortcuts(enabled);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save keyboard shortcuts setting:", message);
      onError(message);
    }
  };

  const handleSetTheme = async (newTheme: ThemePreference) => {
    try {
      await onSetTheme(newTheme);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save theme preference:", message);
      onError(message);
    }
  };

  const handleSaveKey = async () => {
    const trimmed = apiKey.trim();
    if (!trimmed) return;

    setSaving(true);
    setSaved(false);
    try {
      await SetReedAPIKey(trimmed);
      setHasKey(true);
      setApiKey("");
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

  const handleClearKey = async () => {
    setClearing(true);
    setSaved(false);
    try {
      await ClearReedAPIKey();
      setApiKey("");
      setHasKey(false);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to clear API key:", message);
      onError(message);
    } finally {
      setClearing(false);
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
              placeholder={
                hasKey
                  ? "Enter new API key to replace existing key"
                  : "Enter API key"
              }
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
            {hasKey && (
              <Button
                variant="danger"
                size="sm"
                onClick={handleClearKey}
                loading={clearing}
                className="shrink-0"
              >
                Clear
              </Button>
            )}
          </div>
          {hasKey && (
            <p className="mt-1 text-xs text-hw-text-muted">
              Key is stored securely in your OS keychain.
            </p>
          )}
          {!hasKey && (
            <Button
              variant="secondary"
              size="sm"
              className="mt-1"
              onClick={() =>
                Browser.OpenURL(
                  "https://www.reed.co.uk/developers/Jobseeker"
                )
              }
            >
              Obtain a Key
            </Button>
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
                onClick={() => {
                  void handleSetTheme(opt.value);
                }}
                aria-pressed={theme === opt.value}
              >
                {opt.label}
              </Button>
            ))}
          </div>
        </section>

        {/* Keyboard Shortcuts Section */}
        <section>
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-sm font-semibold text-hw-text">
              Keyboard Shortcuts
            </h3>
            <IconButton
              aria-label="Show keyboard shortcuts"
              onClick={() => setShortcutsHelpOpen(true)}
            >
              ?
            </IconButton>
          </div>
          <div className="flex gap-2">
            <Button
              variant={keyboardShortcuts ? "primary" : "secondary"}
              size="sm"
              onClick={() => handleSetKeyboardShortcuts(true)}
              aria-pressed={keyboardShortcuts}
            >
              Enabled
            </Button>
            <Button
              variant={!keyboardShortcuts ? "primary" : "secondary"}
              size="sm"
              onClick={() => handleSetKeyboardShortcuts(false)}
              aria-pressed={!keyboardShortcuts}
            >
              Disabled
            </Button>
          </div>
        </section>
      </div>

      {shortcutsHelpOpen && (
        <ShortcutsHelp onClose={() => setShortcutsHelpOpen(false)} />
      )}
    </div>
  );
}
