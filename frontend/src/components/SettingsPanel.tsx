import { useEffect, useRef, useState, type MutableRefObject } from "react";
import { Browser } from "@wailsio/runtime";
import {
  ClearOpenAIAPIKey,
  ClearReedAPIKey,
  GetLLMModel,
  GetLLMProvider,
  HasOpenAIAPIKey,
  HasReedAPIKey,
  SetKeyboardShortcuts,
  SetLLMModel,
  SetLLMProvider,
  SetOpenAIAPIKey,
  SetReedAPIKey,
} from "../../bindings/hamster-wheel/settingsservice";
import { Button } from "./Button";
import { IconButton } from "./IconButton";
import { Input } from "./Input";
import { ShortcutsHelp } from "./ShortcutsHelp";
import type { ThemePreference } from "../hooks/useTheme";

interface SettingsPanelProps {
  onClose: () => void;
  theme: ThemePreference;
  onSetTheme: (theme: ThemePreference) => Promise<void>;
  onError: (msg: string) => void;
  keyboardShortcuts: boolean;
  onSetKeyboardShortcuts: (enabled: boolean) => void;
}

type SettingsTab = "interface" | "jobs-providers" | "llm-providers";

const tabs: { id: SettingsTab; label: string }[] = [
  { id: "interface", label: "Interface" },
  { id: "jobs-providers", label: "Jobs Providers" },
  { id: "llm-providers", label: "LLM Providers" },
];

const themeOptions: { value: ThemePreference; label: string }[] = [
  { value: "system", label: "System" },
  { value: "dark", label: "Dark" },
  { value: "light", label: "Light" },
];

const llmProviderOptions = [
  { value: "openai", label: "OpenAI" },
  { value: "heuristic_v1", label: "Heuristic (Local)" },
];

const llmModelOptionsByProvider: Record<
  string,
  { value: string; label: string }[]
> = {
  openai: [
    { value: "gpt-4o-mini", label: "gpt-4o-mini" },
    { value: "gpt-4o", label: "gpt-4o" },
    { value: "gpt-4.1-mini", label: "gpt-4.1-mini" },
    { value: "gpt-4.1", label: "gpt-4.1" },
  ],
  heuristic_v1: [{ value: "heuristic_v1", label: "heuristic_v1" }],
};

const selectClasses =
  "w-full rounded bg-hw-bg border border-hw-border text-hw-text text-sm px-2 py-1.5 focus:outline-none focus:border-hw-accent focus-visible:ring-2 focus-visible:ring-hw-accent focus-visible:ring-offset-1 focus-visible:ring-offset-hw-bg transition-colors duration-150";

export function SettingsPanel({
  onClose,
  theme,
  onSetTheme,
  onError,
  keyboardShortcuts,
  onSetKeyboardShortcuts,
}: SettingsPanelProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>("interface");
  const [shortcutsHelpOpen, setShortcutsHelpOpen] = useState(false);

  const [reedAPIKey, setReedAPIKey] = useState("");
  const [reedSaved, setReedSaved] = useState(false);
  const [reedSaving, setReedSaving] = useState(false);
  const [reedClearing, setReedClearing] = useState(false);
  const [hasReedKey, setHasReedKey] = useState(false);

  const [openAIAPIKey, setOpenAIAPIKey] = useState("");
  const [openAISaved, setOpenAISaved] = useState(false);
  const [openAISaving, setOpenAISaving] = useState(false);
  const [openAIClearing, setOpenAIClearing] = useState(false);
  const [hasOpenAIKey, setHasOpenAIKey] = useState(false);

  const [llmProvider, setLLMProviderState] = useState("openai");
  const [llmModel, setLLMModelState] = useState("gpt-4o-mini");
  const [llmConfigSaving, setLLMConfigSaving] = useState(false);
  const [llmConfigSaved, setLLMConfigSaved] = useState(false);

  const reedSavedTimeoutRef = useRef<number | null>(null);
  const openAISavedTimeoutRef = useRef<number | null>(null);
  const llmSavedTimeoutRef = useRef<number | null>(null);

  useEffect(() => {
    let cancelled = false;

    const reportError = (prefix: string, err: unknown) => {
      const message = err instanceof Error ? err.message : String(err);
      console.error(prefix, message);
      if (!cancelled) {
        onError(message);
      }
    };

    const load = async () => {
      try {
        const present = await HasReedAPIKey();
        if (!cancelled) {
          setHasReedKey(Boolean(present));
        }
      } catch (err: unknown) {
        reportError("Failed to load Reed API key:", err);
      }

      try {
        const present = await HasOpenAIAPIKey();
        if (!cancelled) {
          setHasOpenAIKey(Boolean(present));
        }
      } catch (err: unknown) {
        reportError("Failed to load OpenAI API key:", err);
      }

      try {
        const provider = await GetLLMProvider();
        if (!cancelled) {
          if (
            Object.prototype.hasOwnProperty.call(
              llmModelOptionsByProvider,
              provider
            )
          ) {
            setLLMProviderState(provider);
          } else {
            setLLMProviderState("openai");
          }
        }
      } catch (err: unknown) {
        reportError("Failed to load LLM provider:", err);
      }

      try {
        const model = await GetLLMModel();
        if (!cancelled) {
          setLLMModelState(model || "gpt-4o-mini");
        }
      } catch (err: unknown) {
        reportError("Failed to load LLM model:", err);
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [onError]);

  useEffect(() => {
    return () => {
      if (reedSavedTimeoutRef.current !== null) {
        clearTimeout(reedSavedTimeoutRef.current);
      }
      if (openAISavedTimeoutRef.current !== null) {
        clearTimeout(openAISavedTimeoutRef.current);
      }
      if (llmSavedTimeoutRef.current !== null) {
        clearTimeout(llmSavedTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    const validModels =
      llmModelOptionsByProvider[llmProvider] ?? llmModelOptionsByProvider.openai;
    if (!validModels.some((option) => option.value === llmModel)) {
      setLLMModelState(validModels[0].value);
    }
  }, [llmModel, llmProvider]);

  const setTimedSavedState = (
    setter: (saved: boolean) => void,
    timeoutRef: MutableRefObject<number | null>
  ) => {
    setter(true);
    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current);
    }
    timeoutRef.current = window.setTimeout(() => {
      setter(false);
      timeoutRef.current = null;
    }, 2000);
  };

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

  const handleSaveReedKey = async () => {
    const trimmed = reedAPIKey.trim();
    if (!trimmed) return;

    setReedSaving(true);
    setReedSaved(false);
    try {
      await SetReedAPIKey(trimmed);
      setHasReedKey(true);
      setReedAPIKey("");
      setTimedSavedState(setReedSaved, reedSavedTimeoutRef);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save Reed API key:", message);
      onError(message);
    } finally {
      setReedSaving(false);
    }
  };

  const handleClearReedKey = async () => {
    setReedClearing(true);
    setReedSaved(false);
    try {
      await ClearReedAPIKey();
      setReedAPIKey("");
      setHasReedKey(false);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to clear Reed API key:", message);
      onError(message);
    } finally {
      setReedClearing(false);
    }
  };

  const handleSaveOpenAIKey = async () => {
    const trimmed = openAIAPIKey.trim();
    if (!trimmed) return;

    setOpenAISaving(true);
    setOpenAISaved(false);
    try {
      await SetOpenAIAPIKey(trimmed);
      setHasOpenAIKey(true);
      setOpenAIAPIKey("");
      setTimedSavedState(setOpenAISaved, openAISavedTimeoutRef);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save OpenAI API key:", message);
      onError(message);
    } finally {
      setOpenAISaving(false);
    }
  };

  const handleClearOpenAIKey = async () => {
    setOpenAIClearing(true);
    setOpenAISaved(false);
    try {
      await ClearOpenAIAPIKey();
      setOpenAIAPIKey("");
      setHasOpenAIKey(false);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to clear OpenAI API key:", message);
      onError(message);
    } finally {
      setOpenAIClearing(false);
    }
  };

  const handleSaveLLMConfig = async () => {
    const trimmedModel = llmModel.trim();
    if (!trimmedModel) {
      onError("LLM model is required");
      return;
    }

    setLLMConfigSaving(true);
    setLLMConfigSaved(false);
    try {
      await SetLLMProvider(llmProvider);
      await SetLLMModel(trimmedModel);
      setLLMModelState(trimmedModel);
      setTimedSavedState(setLLMConfigSaved, llmSavedTimeoutRef);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save LLM provider settings:", message);
      onError(message);
    } finally {
      setLLMConfigSaving(false);
    }
  };

  const modelOptions =
    llmModelOptionsByProvider[llmProvider] ?? llmModelOptionsByProvider.openai;

  return (
    <div
      className="absolute top-0 right-0 bottom-0 w-[400px] bg-hw-surface border-l border-hw-border shadow-lg z-10 flex flex-col overflow-hidden transition-opacity duration-200"
      role="dialog"
      aria-label="Settings"
    >
      <div className="flex items-center justify-between px-4 py-3 border-b border-hw-border shrink-0">
        <h2 className="text-lg font-bold text-hw-text">Settings</h2>
        <IconButton aria-label="Close settings" onClick={onClose}>
          ✕
        </IconButton>
      </div>

      <div className="shrink-0 px-4 pt-3">
        <div role="tablist" aria-label="Settings sections" className="flex gap-2">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              role="tab"
              id={`settings-tab-${tab.id}`}
              aria-controls={`settings-panel-${tab.id}`}
              aria-selected={activeTab === tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`rounded px-3 py-1.5 text-xs font-medium border transition-colors duration-150 ${
                activeTab === tab.id
                  ? "bg-hw-accent border-hw-accent text-white"
                  : "bg-hw-bg border-hw-border text-hw-text hover:bg-hw-surface-hover"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        <div
          role="tabpanel"
          id="settings-panel-interface"
          aria-labelledby="settings-tab-interface"
          hidden={activeTab !== "interface"}
          className="space-y-6"
        >
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

        <div
          role="tabpanel"
          id="settings-panel-jobs-providers"
          aria-labelledby="settings-tab-jobs-providers"
          hidden={activeTab !== "jobs-providers"}
        >
          <section>
            <h3 className="text-sm font-semibold text-hw-text mb-2">
              Reed API Key
            </h3>
            <div className="flex gap-2">
              <Input
                type="password"
                size="sm"
                value={reedAPIKey}
                onChange={(e) => {
                  setReedAPIKey(e.target.value);
                  setReedSaved(false);
                }}
                placeholder={
                  hasReedKey
                    ? "Enter new API key to replace existing key"
                    : "Enter API key"
                }
                className="flex-1 min-w-0"
                aria-label="Reed API Key"
              />
              <Button
                variant="primary"
                size="sm"
                onClick={handleSaveReedKey}
                disabled={!reedAPIKey.trim()}
                loading={reedSaving}
                className="shrink-0"
              >
                {reedSaved ? "Saved" : "Save"}
              </Button>
              {hasReedKey && (
                <Button
                  variant="danger"
                  size="sm"
                  onClick={handleClearReedKey}
                  loading={reedClearing}
                  className="shrink-0"
                >
                  Clear
                </Button>
              )}
            </div>
            {hasReedKey ? (
              <p className="mt-1 text-xs text-hw-text-muted">
                Key is stored securely in your OS keychain.
              </p>
            ) : (
              <Button
                variant="secondary"
                size="sm"
                className="mt-1"
                onClick={() =>
                  Browser.OpenURL("https://www.reed.co.uk/developers/Jobseeker")
                }
              >
                Obtain a Key
              </Button>
            )}
          </section>
        </div>

        <div
          role="tabpanel"
          id="settings-panel-llm-providers"
          aria-labelledby="settings-tab-llm-providers"
          hidden={activeTab !== "llm-providers"}
          className="space-y-6"
        >
          <section>
            <h3 className="text-sm font-semibold text-hw-text mb-2">
              Provider Configuration
            </h3>
            <div className="space-y-3">
              <div>
                <label
                  htmlFor="llm-provider"
                  className="block text-xs text-hw-text-muted mb-1"
                >
                  Active provider
                </label>
                <select
                  id="llm-provider"
                  aria-label="LLM Provider"
                  value={llmProvider}
                  onChange={(e) => {
                    const nextProvider = e.target.value;
                    const nextModelOptions =
                      llmModelOptionsByProvider[nextProvider] ??
                      llmModelOptionsByProvider.openai;
                    setLLMProviderState(nextProvider);
                    setLLMModelState(nextModelOptions[0].value);
                    setLLMConfigSaved(false);
                  }}
                  className={selectClasses}
                >
                  {llmProviderOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label
                  htmlFor="llm-model"
                  className="block text-xs text-hw-text-muted mb-1"
                >
                  Model
                </label>
                <select
                  id="llm-model"
                  aria-label="LLM Model"
                  value={llmModel}
                  onChange={(e) => {
                    setLLMModelState(e.target.value);
                    setLLMConfigSaved(false);
                  }}
                  className={selectClasses}
                >
                  {modelOptions.map((model) => (
                    <option key={model.value} value={model.value}>
                      {model.label}
                    </option>
                  ))}
                </select>
              </div>

              <Button
                variant="primary"
                size="sm"
                onClick={handleSaveLLMConfig}
                loading={llmConfigSaving}
              >
                {llmConfigSaved ? "Saved" : "Save configuration"}
              </Button>
            </div>
          </section>

          <section>
            <h3 className="text-sm font-semibold text-hw-text mb-2">
              OpenAI API Key
            </h3>
            <div className="flex gap-2">
              <Input
                type="password"
                size="sm"
                value={openAIAPIKey}
                onChange={(e) => {
                  setOpenAIAPIKey(e.target.value);
                  setOpenAISaved(false);
                }}
                placeholder={
                  hasOpenAIKey
                    ? "Enter new API key to replace existing key"
                    : "Enter API key"
                }
                className="flex-1 min-w-0"
                aria-label="OpenAI API Key"
              />
              <Button
                variant="primary"
                size="sm"
                onClick={handleSaveOpenAIKey}
                disabled={!openAIAPIKey.trim()}
                loading={openAISaving}
                className="shrink-0"
              >
                {openAISaved ? "Saved" : "Save"}
              </Button>
              {hasOpenAIKey && (
                <Button
                  variant="danger"
                  size="sm"
                  onClick={handleClearOpenAIKey}
                  loading={openAIClearing}
                  className="shrink-0"
                >
                  Clear
                </Button>
              )}
            </div>
            {hasOpenAIKey ? (
              <p className="mt-1 text-xs text-hw-text-muted">
                Key is stored securely in your OS keychain.
              </p>
            ) : (
              <Button
                variant="secondary"
                size="sm"
                className="mt-1"
                onClick={() => Browser.OpenURL("https://platform.openai.com/api-keys")}
              >
                Obtain an API Key
              </Button>
            )}
          </section>
        </div>
      </div>

      {shortcutsHelpOpen && (
        <ShortcutsHelp onClose={() => setShortcutsHelpOpen(false)} />
      )}
    </div>
  );
}
