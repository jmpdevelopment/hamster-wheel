import { useEffect, useRef, useState, type MutableRefObject } from "react";
import { Browser, Dialogs } from "@wailsio/runtime";
import {
  ClearOpenAIAPIKey,
  ClearReedAPIKey,
  GetAutoMatchEnabled,
  GetAutoMatchLimit,
  GetCVPath,
  GetLLMBaseURL,
  GetLLMMode,
  GetLLMModel,
  GetLLMProvider,
  GetLocalRuntimeModels,
  GetLocalRuntimeModel,
  GetLocalRuntimePullProgress,
  GetLocalRuntimeStatus,
  HasOpenAIAPIKey,
  HasReedAPIKey,
  PullLocalRuntimeModel,
  SetCVPath,
  SetKeyboardShortcuts,
  SetLLMBaseURL,
  SetLLMMode,
  SetLLMModel,
  SetLLMProvider,
  SetLocalRuntimeEngine,
  SetLocalRuntimeModel,
  SetOpenAIAPIKey,
  SetAutoMatchEnabled,
  SetAutoMatchLimit,
  SetReedAPIKey,
  StartLocalRuntime,
  StopLocalRuntime,
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

const cloudModelOptions = [
  { value: "gpt-4o-mini", label: "gpt-4o-mini" },
  { value: "gpt-4o", label: "gpt-4o" },
  { value: "gpt-4.1-mini", label: "gpt-4.1-mini" },
  { value: "gpt-4.1", label: "gpt-4.1" },
];

const llmModelOptionsByProvider: Record<
  string,
  { value: string; label: string }[]
> = {
  openai: cloudModelOptions,
  heuristic_v1: [{ value: "heuristic_v1", label: "heuristic_v1" }],
};

type LLMMode = "cloud" | "local" | "advanced";

const localModelName = "llama3.1:8b";
const localModelEstimatedBytes = 4_900_000_000;
const llamaLicenseURL = "https://www.llama.com/llama3_1/license/";
const llamaUsePolicyURL = "https://www.llama.com/llama3_1/use-policy/";

function formatGiB(bytes: number): string {
  return `${(bytes / (1024 ** 3)).toFixed(1)} GiB`;
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  const precision = value >= 10 || unitIndex === 0 ? 0 : 1;
  return `${value.toFixed(precision)} ${units[unitIndex]}`;
}

interface LocalPullProgressState {
  active: boolean;
  model: string;
  status: string;
  message: string;
  totalBytes: number;
  completedBytes: number;
  percent: number | null;
  ready: boolean;
}

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

  const [llmMode, setLLMModeState] = useState<LLMMode>("cloud");
  const [llmProvider, setLLMProviderState] = useState("openai");
  const [llmModel, setLLMModelState] = useState("gpt-4o-mini");
  const [llmBaseURL, setLLMBaseURLState] = useState("");
  const [localRuntimeModel, setLocalRuntimeModelState] = useState(localModelName);
  const [localRuntimeStatus, setLocalRuntimeStatus] = useState("unknown");
  const [localRuntimeMessage, setLocalRuntimeMessage] = useState("");
  const [localRuntimeStartedByApp, setLocalRuntimeStartedByApp] = useState(false);
  const [localRuntimeReady, setLocalRuntimeReady] = useState(false);
  const [localModelInstalled, setLocalModelInstalled] = useState(false);
  const [localRuntimeRefreshing, setLocalRuntimeRefreshing] = useState(false);
  const [localRuntimeStarting, setLocalRuntimeStarting] = useState(false);
  const [localRuntimeStopping, setLocalRuntimeStopping] = useState(false);
  const [localModelPulling, setLocalModelPulling] = useState(false);
  const [localPullProgress, setLocalPullProgress] =
    useState<LocalPullProgressState>({
      active: false,
      model: "",
      status: "",
      message: "",
      totalBytes: 0,
      completedBytes: 0,
      percent: null,
      ready: false,
    });
  const [llmConfigSaving, setLLMConfigSaving] = useState(false);
  const [llmConfigSaved, setLLMConfigSaved] = useState(false);
  const [autoMatchEnabled, setAutoMatchEnabledState] = useState(true);
  const [autoMatchLimit, setAutoMatchLimitState] = useState("0");
  const [autoMatchSaving, setAutoMatchSaving] = useState(false);
  const [autoMatchSaved, setAutoMatchSaved] = useState(false);
  const [cvPath, setCVPathState] = useState("");
  const [cvPathSaving, setCVPathSaving] = useState(false);
  const [cvPathSaved, setCVPathSaved] = useState(false);

  const reedSavedTimeoutRef = useRef<number | null>(null);
  const openAISavedTimeoutRef = useRef<number | null>(null);
  const llmSavedTimeoutRef = useRef<number | null>(null);
  const autoMatchSavedTimeoutRef = useRef<number | null>(null);
  const cvSavedTimeoutRef = useRef<number | null>(null);

  const refreshLocalPullProgress = async (reportErrors = true) => {
    try {
      const progress = await GetLocalRuntimePullProgress();
      const totalBytes = Number(progress.totalBytes || 0);
      const completedBytes = Number(progress.completedBytes || 0);
      const rawPercent = Number(progress.percent);
      const percent =
        Number.isFinite(rawPercent) && rawPercent >= 0
          ? Math.min(100, Math.max(0, rawPercent))
          : null;
      const active = Boolean(progress.active);

      setLocalPullProgress({
        active,
        model: String(progress.model || ""),
        status: String(progress.status || ""),
        message: String(progress.message || ""),
        totalBytes: Number.isFinite(totalBytes) && totalBytes > 0 ? totalBytes : 0,
        completedBytes:
          Number.isFinite(completedBytes) && completedBytes > 0
            ? completedBytes
            : 0,
        percent,
        ready: Boolean(progress.ready),
      });
      setLocalModelPulling(active);
    } catch (err: unknown) {
      if (reportErrors) {
        const message = err instanceof Error ? err.message : String(err);
        console.error("Failed to refresh local runtime pull progress:", message);
        onError(message);
      }
    }
  };

  const refreshLocalRuntime = async (reportErrors = true) => {
    setLocalRuntimeRefreshing(true);
    try {
      await refreshLocalPullProgress(reportErrors);

      const snapshot = await GetLocalRuntimeStatus();
      setLocalRuntimeStatus(String(snapshot.status || "unknown"));
      setLocalRuntimeMessage(String(snapshot.message || ""));
      setLocalRuntimeStartedByApp(Boolean(snapshot.startedByApp));

      const ready = String(snapshot.status || "") === "ready";
      setLocalRuntimeReady(ready);
      if (!ready) {
        setLocalModelInstalled(false);
        return;
      }

      const catalog = await GetLocalRuntimeModels();
      const installed = Array.isArray(catalog.installed)
        ? catalog.installed.some(
            (model: { name?: string }) => String(model?.name || "") === localModelName
          )
        : false;
      setLocalModelInstalled(installed);
    } catch (err: unknown) {
      setLocalRuntimeReady(false);
      setLocalModelInstalled(false);
      const message = err instanceof Error ? err.message : String(err);
      if (reportErrors) {
        console.error("Failed to refresh local runtime state:", message);
        onError(message);
      }
    } finally {
      setLocalRuntimeRefreshing(false);
    }
  };

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
        const mode = await GetLLMMode();
        if (!cancelled) {
          if (mode === "cloud" || mode === "local" || mode === "advanced") {
            setLLMModeState(mode);
          } else {
            setLLMModeState("cloud");
          }
        }
      } catch (err: unknown) {
        reportError("Failed to load LLM mode:", err);
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

      try {
        const baseURL = await GetLLMBaseURL();
        if (!cancelled) {
          setLLMBaseURLState(baseURL || "");
        }
      } catch (err: unknown) {
        reportError("Failed to load LLM base URL:", err);
      }

      try {
        const localModel = await GetLocalRuntimeModel();
        if (!cancelled) {
          setLocalRuntimeModelState(
            localModel === localModelName ? localModel : localModelName
          );
        }
      } catch (err: unknown) {
        reportError("Failed to load local runtime model:", err);
      }

      try {
        const enabled = await GetAutoMatchEnabled();
        if (!cancelled) {
          setAutoMatchEnabledState(Boolean(enabled));
        }
      } catch (err: unknown) {
        reportError("Failed to load auto match enabled setting:", err);
      }

      try {
        const limit = await GetAutoMatchLimit();
        if (!cancelled) {
          const normalizedLimit =
            Number.isFinite(limit) && Number(limit) >= 0
              ? Math.floor(Number(limit))
              : 0;
          setAutoMatchLimitState(String(normalizedLimit));
        }
      } catch (err: unknown) {
        reportError("Failed to load auto match limit setting:", err);
      }

      try {
        const storedPath = await GetCVPath();
        if (!cancelled) {
          setCVPathState(storedPath || "");
        }
      } catch (err: unknown) {
        reportError("Failed to load CV path:", err);
      }

      if (!cancelled) {
        await refreshLocalRuntime(false);
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
      if (autoMatchSavedTimeoutRef.current !== null) {
        clearTimeout(autoMatchSavedTimeoutRef.current);
      }
      if (cvSavedTimeoutRef.current !== null) {
        clearTimeout(cvSavedTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (llmMode === "local") {
      void refreshLocalRuntime(false);
    }
  }, [llmMode]);

  useEffect(() => {
    if (llmMode !== "local") {
      return;
    }

    let cancelled = false;
    const refresh = async () => {
      if (cancelled) {
        return;
      }
      await refreshLocalPullProgress(false);
    };

    void refresh();
    const intervalID = window.setInterval(() => {
      void refresh();
    }, 1000);

    return () => {
      cancelled = true;
      clearInterval(intervalID);
    };
  }, [llmMode]);

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

  const localPullInFlight = localModelPulling || localPullProgress.active;
  const showLocalPullStatus =
    localPullProgress.active ||
    (!localPullProgress.ready &&
      (Boolean(localPullProgress.status) || Boolean(localPullProgress.message)));
  const hasLocalPullPercent = localPullProgress.percent !== null;
  const localPullByteLabel =
    localPullProgress.totalBytes > 0
      ? `${formatBytes(localPullProgress.completedBytes)} / ${formatBytes(localPullProgress.totalBytes)}`
      : "";

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

  const handleSaveCloudConfig = async () => {
    const trimmedModel = llmModel.trim();
    if (!trimmedModel) {
      onError("Cloud model is required");
      return;
    }

    setLLMConfigSaving(true);
    setLLMConfigSaved(false);
    try {
      await SetLLMMode("cloud");
      await SetLLMProvider("openai");
      await SetLLMModel(trimmedModel);
      await SetLLMBaseURL("");
      setLLMModeState("cloud");
      setLLMProviderState("openai");
      setLLMModelState(trimmedModel);
      setLLMBaseURLState("");
      setTimedSavedState(setLLMConfigSaved, llmSavedTimeoutRef);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save cloud LLM settings:", message);
      onError(message);
    } finally {
      setLLMConfigSaving(false);
    }
  };

  const handleSaveLocalConfig = async () => {
    const trimmedModel = localRuntimeModel.trim();
    if (trimmedModel !== localModelName) {
      onError(`Local mode currently supports ${localModelName} only.`);
      return;
    }
    if (!localRuntimeReady) {
      onError("Start local runtime before enabling Local mode.");
      return;
    }
    if (!localModelInstalled) {
      onError(`Download ${localModelName} before enabling Local mode.`);
      return;
    }

    setLLMConfigSaving(true);
    setLLMConfigSaved(false);
    try {
      await SetLLMMode("local");
      await SetLocalRuntimeEngine("ollama");
      await SetLocalRuntimeModel(localModelName);
      setLLMModeState("local");
      setLocalRuntimeModelState(localModelName);
      setTimedSavedState(setLLMConfigSaved, llmSavedTimeoutRef);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save local runtime settings:", message);
      onError(message);
    } finally {
      setLLMConfigSaving(false);
    }
  };

  const handleStartLocalRuntime = async () => {
    setLocalRuntimeStarting(true);
    try {
      await StartLocalRuntime();
      await refreshLocalRuntime(false);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to start local runtime:", message);
      onError(message);
    } finally {
      setLocalRuntimeStarting(false);
    }
  };

  const handleStopLocalRuntime = async () => {
    setLocalRuntimeStopping(true);
    try {
      await StopLocalRuntime();
      await refreshLocalRuntime(false);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to stop local runtime:", message);
      onError(message);
    } finally {
      setLocalRuntimeStopping(false);
    }
  };

  const handlePullLocalModel = async () => {
    if (localPullInFlight) {
      onError(`Download for ${localModelName} is already in progress.`);
      return;
    }
    setLocalModelPulling(true);
    try {
      await PullLocalRuntimeModel(localModelName);
      await refreshLocalRuntime(false);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to pull local runtime model:", message);
      onError(message);
    } finally {
      setLocalModelPulling(false);
      await refreshLocalPullProgress(false);
    }
  };

  const handleSaveAdvancedConfig = async () => {
    const trimmedModel = llmModel.trim();
    if (!trimmedModel) {
      onError("LLM model is required");
      return;
    }

    const trimmedBaseURL = llmBaseURL.trim();

    setLLMConfigSaving(true);
    setLLMConfigSaved(false);
    try {
      await SetLLMMode("advanced");
      await SetLLMProvider(llmProvider);
      await SetLLMModel(trimmedModel);
      await SetLLMBaseURL(trimmedBaseURL);
      setLLMModeState("advanced");
      setLLMModelState(trimmedModel);
      setLLMBaseURLState(trimmedBaseURL);
      setTimedSavedState(setLLMConfigSaved, llmSavedTimeoutRef);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save advanced LLM settings:", message);
      onError(message);
    } finally {
      setLLMConfigSaving(false);
    }
  };

  const handleSaveAutoMatchSettings = async () => {
    const trimmedLimit = autoMatchLimit.trim();
    const parsedLimit = Number(trimmedLimit);
    if (
      !Number.isFinite(parsedLimit) ||
      !Number.isInteger(parsedLimit) ||
      parsedLimit < 0
    ) {
      onError("Auto-match limit must be a whole number greater than or equal to 0.");
      return;
    }

    setAutoMatchSaving(true);
    setAutoMatchSaved(false);
    try {
      await SetAutoMatchEnabled(autoMatchEnabled);
      await SetAutoMatchLimit(parsedLimit);
      setAutoMatchLimitState(String(parsedLimit));
      setTimedSavedState(setAutoMatchSaved, autoMatchSavedTimeoutRef);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save auto match settings:", message);
      onError(message);
    } finally {
      setAutoMatchSaving(false);
    }
  };

  const handleBrowseCVPath = async () => {
    try {
      const picked = await Dialogs.OpenFile({
        Title: "Select CV File",
        Message: "Choose a CV file (PDF or plain text) for matching context.",
        CanChooseFiles: true,
        CanChooseDirectories: false,
        AllowsMultipleSelection: false,
        Filters: [
          { DisplayName: "PDF files", Pattern: "*.pdf" },
          { DisplayName: "Text files", Pattern: "*.txt;*.md;*.markdown" },
          { DisplayName: "All files", Pattern: "*" },
        ],
      });
      const nextPath = Array.isArray(picked) ? picked[0] ?? "" : picked;
      if (nextPath) {
        setCVPathState(nextPath);
        setCVPathSaved(false);
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to browse CV path:", message);
      onError(message);
    }
  };

  const handleSaveCVPath = async () => {
    const trimmed = cvPath.trim();
    if (!trimmed) return;

    setCVPathSaving(true);
    setCVPathSaved(false);
    try {
      await SetCVPath(trimmed);
      setCVPathState(trimmed);
      setTimedSavedState(setCVPathSaved, cvSavedTimeoutRef);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to save CV path:", message);
      onError(message);
    } finally {
      setCVPathSaving(false);
    }
  };

  const handleClearCVPath = async () => {
    setCVPathSaving(true);
    setCVPathSaved(false);
    try {
      await SetCVPath("");
      setCVPathState("");
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      console.error("Failed to clear CV path:", message);
      onError(message);
    } finally {
      setCVPathSaving(false);
    }
  };

  const effectiveCloudModelOptions =
    cloudModelOptions.some((option) => option.value === llmModel) || !llmModel.trim()
      ? cloudModelOptions
      : [{ value: llmModel, label: `${llmModel} (Current)` }, ...cloudModelOptions];

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
              Auto Match Calculations
            </h3>
            <p className="text-xs text-hw-text-muted mb-2">
              Control automatic match scoring for newly discovered jobs. Manual
              `Recalculate score` remains available per job.
            </p>
            <div className="space-y-3">
              <div className="flex gap-2">
                <Button
                  variant={autoMatchEnabled ? "primary" : "secondary"}
                  size="sm"
                  onClick={() => {
                    setAutoMatchEnabledState(true);
                    setAutoMatchSaved(false);
                  }}
                  aria-pressed={autoMatchEnabled}
                >
                  Enabled
                </Button>
                <Button
                  variant={!autoMatchEnabled ? "primary" : "secondary"}
                  size="sm"
                  onClick={() => {
                    setAutoMatchEnabledState(false);
                    setAutoMatchSaved(false);
                  }}
                  aria-pressed={!autoMatchEnabled}
                >
                  Disabled
                </Button>
              </div>
              <div>
                <label
                  htmlFor="auto-match-limit"
                  className="block text-xs text-hw-text-muted mb-1"
                >
                  Auto-match limit per poll cycle (0 = no limit)
                </label>
                <Input
                  id="auto-match-limit"
                  aria-label="Auto Match Limit"
                  type="number"
                  size="sm"
                  min={0}
                  step={1}
                  value={autoMatchLimit}
                  onChange={(e) => {
                    setAutoMatchLimitState(e.target.value);
                    setAutoMatchSaved(false);
                  }}
                />
                <p className="mt-1 text-xs text-hw-text-muted">
                  Use a smaller limit to reduce cloud token costs and local CPU/RAM pressure.
                </p>
              </div>
              <Button
                variant="primary"
                size="sm"
                onClick={handleSaveAutoMatchSettings}
                loading={autoMatchSaving}
              >
                {autoMatchSaved ? "Saved" : "Save auto-match settings"}
              </Button>
            </div>
          </section>

          <section>
            <h3 className="text-sm font-semibold text-hw-text mb-2">
              CV Context for Matching
            </h3>
            <p className="text-xs text-hw-text-muted mb-2">
              Add a CV file so match scoring uses your profile context.
            </p>
            <div className="space-y-2">
              <Input
                type="text"
                size="sm"
                value={cvPath}
                onChange={(e) => {
                  setCVPathState(e.target.value);
                  setCVPathSaved(false);
                }}
                placeholder="/path/to/cv.txt"
                className="w-full"
                aria-label="CV File Path"
              />
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    void handleBrowseCVPath();
                  }}
                >
                  Browse
                </Button>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleSaveCVPath}
                  disabled={!cvPath.trim()}
                  loading={cvPathSaving}
                >
                  {cvPathSaved ? "Saved" : "Save CV path"}
                </Button>
                {cvPath.trim() && (
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={handleClearCVPath}
                    loading={cvPathSaving}
                  >
                    Clear
                  </Button>
                )}
              </div>
              <p className="text-xs text-hw-text-muted">
                Supported CV formats: PDF and plain text (`.txt`, `.md`,
                `.markdown`). Unsupported formats are rejected on save.
              </p>
            </div>
          </section>

          <section>
            <h3 className="text-sm font-semibold text-hw-text mb-2">
              Matching Mode
            </h3>
            <p className="text-xs text-hw-text-muted mb-2">
              Cloud and Local are guided modes. Advanced exposes manual endpoint
              settings.
            </p>
            <div
              role="radiogroup"
              aria-label="LLM Mode"
              className="grid grid-cols-3 gap-2"
            >
              {[
                { value: "cloud", label: "Cloud" },
                { value: "local", label: "Local" },
                { value: "advanced", label: "Advanced" },
              ].map((modeOption) => (
                <button
                  key={modeOption.value}
                  type="button"
                  role="radio"
                  aria-checked={llmMode === modeOption.value}
                  onClick={() => {
                    setLLMModeState(modeOption.value as LLMMode);
                    setLLMConfigSaved(false);
                  }}
                  className={`rounded px-3 py-2 text-xs font-medium border text-left transition-colors duration-150 ${
                    llmMode === modeOption.value
                      ? "bg-hw-accent border-hw-accent text-white"
                      : "bg-hw-bg border-hw-border text-hw-text hover:bg-hw-surface-hover"
                  }`}
                >
                  {modeOption.label}
                </button>
              ))}
            </div>
          </section>

          {llmMode === "cloud" && (
            <section>
              <h3 className="text-sm font-semibold text-hw-text mb-2">
                Cloud Configuration
              </h3>
              <p className="text-xs text-hw-text-muted mb-2">
                Use OpenAI hosted models. No endpoint setup required.
              </p>
              <div className="space-y-3">
                <div>
                  <label
                    htmlFor="cloud-llm-model"
                    className="block text-xs text-hw-text-muted mb-1"
                  >
                    Cloud model
                  </label>
                  <select
                    id="cloud-llm-model"
                    aria-label="Cloud LLM Model"
                    value={llmModel}
                    onChange={(e) => {
                      setLLMModelState(e.target.value);
                      setLLMConfigSaved(false);
                    }}
                    className={selectClasses}
                  >
                    {effectiveCloudModelOptions.map((model) => (
                      <option key={model.value} value={model.value}>
                        {model.label}
                      </option>
                    ))}
                  </select>
                </div>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleSaveCloudConfig}
                  loading={llmConfigSaving}
                >
                  {llmConfigSaved ? "Saved" : "Save cloud settings"}
                </Button>
              </div>
            </section>
          )}

          {llmMode === "local" && (
            <section>
              <h3 className="text-sm font-semibold text-hw-text mb-2">
                Local Configuration
              </h3>
              <p className="text-xs text-hw-text-muted mb-2">
                Local setup uses Ollama + {localModelName}.
              </p>
              <p className="text-xs text-hw-text-muted mb-2">
                After installing Ollama, open it once so the local runtime is available.
              </p>
              <p className="text-xs text-hw-text-muted mb-2">
                Step 1: Start runtime. Step 2: Download Llama.
              </p>
              <div className="rounded border border-hw-border bg-hw-bg px-3 py-2">
                <p className="text-xs font-semibold text-hw-text">
                  System requirements (Llama 3.1 8B local mode)
                </p>
                <p className="mt-1 text-xs text-hw-text-muted">
                  Minimum: 16 GB RAM, modern 4+ core CPU, ~8 GB free disk.
                </p>
                <p className="mt-1 text-xs text-hw-text-muted">
                  Recommended: 24 GB+ RAM, Apple Silicon M2/M3 (or equivalent), SSD, and plugged-in power.
                </p>
                <p className="mt-1 text-xs text-hw-text-muted">
                  Impact while running: higher CPU/RAM usage, increased battery drain, and slower system responsiveness during matching.
                </p>
              </div>
              <div className="space-y-3">
                <div>
                  <label
                    htmlFor="local-runtime-model"
                    className="block text-xs text-hw-text-muted mb-1"
                  >
                    Local model
                  </label>
                  <Input
                    id="local-runtime-model"
                    aria-label="Local Runtime Model"
                    type="text"
                    size="sm"
                    value={localRuntimeModel}
                    readOnly
                  />
                </div>
                <div className="rounded border border-hw-border bg-hw-bg px-3 py-2">
                  <p className="text-xs text-hw-text">
                    Runtime status:{" "}
                    <span className="font-semibold">{localRuntimeStatus}</span>
                  </p>
                  {localRuntimeMessage && (
                    <p className="mt-1 text-xs text-hw-text-muted">
                      {localRuntimeMessage}
                    </p>
                  )}
                  <p className="mt-1 text-xs text-hw-text-muted">
                    Llama installed: {localModelInstalled ? "Yes" : "No"}
                  </p>
                  <p className="mt-1 text-xs text-hw-text-muted">
                    Approx download size: {formatGiB(localModelEstimatedBytes)}
                  </p>
                </div>
                {showLocalPullStatus && (
                  <div className="rounded border border-hw-border bg-hw-bg px-3 py-2">
                    <p className="text-xs text-hw-text">
                      Download status:{" "}
                      <span className="font-semibold">
                        {localPullProgress.active
                          ? "in_progress"
                          : localPullProgress.ready
                            ? "completed"
                            : localPullProgress.status || "idle"}
                      </span>
                    </p>
                    {localPullProgress.message && (
                      <p className="mt-1 text-xs text-hw-text-muted">
                        {localPullProgress.message}
                      </p>
                    )}
                    {hasLocalPullPercent && (
                      <div className="mt-2 space-y-1">
                        <div
                          className="h-1.5 w-full overflow-hidden rounded bg-hw-border"
                          role="progressbar"
                          aria-label="Llama download progress"
                          aria-valuemin={0}
                          aria-valuemax={100}
                          aria-valuenow={Math.round(localPullProgress.percent ?? 0)}
                        >
                          <div
                            className="h-full bg-hw-accent transition-[width] duration-500"
                            style={{ width: `${localPullProgress.percent ?? 0}%` }}
                          />
                        </div>
                        <p className="text-xs text-hw-text-muted">
                          {(localPullProgress.percent ?? 0).toFixed(1)}%
                          {localPullByteLabel ? ` (${localPullByteLabel})` : ""}
                        </p>
                      </div>
                    )}
                  </div>
                )}
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => {
                      void refreshLocalRuntime(true);
                    }}
                    loading={localRuntimeRefreshing}
                  >
                    Refresh status
                  </Button>
                  {localRuntimeStatus === "not_installed" && (
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => Browser.OpenURL("https://ollama.com/download")}
                    >
                      Install Ollama
                    </Button>
                  )}
                  {localRuntimeStatus !== "ready" &&
                    localRuntimeStatus !== "not_installed" && (
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={handleStartLocalRuntime}
                        loading={localRuntimeStarting}
                        disabled={localPullInFlight}
                      >
                        Start runtime
                      </Button>
                    )}
                  {localRuntimeStatus === "ready" && localRuntimeStartedByApp && (
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={handleStopLocalRuntime}
                      loading={localRuntimeStopping}
                      disabled={localPullInFlight}
                    >
                      Stop runtime
                    </Button>
                  )}
                  {localRuntimeStatus === "ready" &&
                    (!localModelInstalled || localPullInFlight) && (
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={handlePullLocalModel}
                        loading={localPullInFlight}
                        disabled={localPullInFlight || localModelInstalled}
                      >
                        {localPullInFlight ? "Downloading Llama..." : "Download Llama"}
                      </Button>
                    )}
                </div>
                <div className="rounded border border-hw-border bg-hw-bg px-3 py-2">
                  <p className="text-xs font-semibold text-hw-text">
                    Built with Llama
                  </p>
                  <p className="mt-1 text-xs text-hw-text-muted">
                    By downloading and using Llama, you must comply with Meta&apos;s
                    Llama Community License and Acceptable Use Policy.
                  </p>
                  <div className="mt-2 flex gap-2">
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => Browser.OpenURL(llamaLicenseURL)}
                    >
                      View license
                    </Button>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => Browser.OpenURL(llamaUsePolicyURL)}
                    >
                      View use policy
                    </Button>
                  </div>
                </div>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleSaveLocalConfig}
                  loading={llmConfigSaving}
                  disabled={!localRuntimeReady || !localModelInstalled}
                >
                  {llmConfigSaved ? "Saved" : "Save local settings"}
                </Button>
              </div>
            </section>
          )}

          {llmMode === "advanced" && (
            <section>
              <h3 className="text-sm font-semibold text-hw-text mb-2">
                Advanced Configuration
              </h3>
              <p className="text-xs text-hw-text-muted mb-2">
                Manual OpenAI-compatible endpoint and model overrides.
              </p>
              <div className="space-y-3">
                <div>
                  <label
                    htmlFor="advanced-llm-provider"
                    className="block text-xs text-hw-text-muted mb-1"
                  >
                    Active provider
                  </label>
                  <select
                    id="advanced-llm-provider"
                    aria-label="LLM Provider"
                    value={llmProvider}
                    onChange={(e) => {
                      const nextProvider = e.target.value;
                      const nextModelOptions =
                        llmModelOptionsByProvider[nextProvider] ??
                        llmModelOptionsByProvider.openai;
                      setLLMProviderState(nextProvider);
                      if (!llmModel.trim()) {
                        setLLMModelState(nextModelOptions[0].value);
                      }
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
                    htmlFor="advanced-llm-model"
                    className="block text-xs text-hw-text-muted mb-1"
                  >
                    Model
                  </label>
                  <Input
                    id="advanced-llm-model"
                    type="text"
                    size="sm"
                    value={llmModel}
                    onChange={(e) => {
                      setLLMModelState(e.target.value);
                      setLLMConfigSaved(false);
                    }}
                    placeholder="gpt-4o-mini"
                    aria-label="LLM Model"
                  />
                </div>

                {llmProvider === "openai" && (
                  <div>
                    <label
                      htmlFor="advanced-llm-base-url"
                      className="block text-xs text-hw-text-muted mb-1"
                    >
                      Base URL (optional)
                    </label>
                    <Input
                      id="advanced-llm-base-url"
                      type="text"
                      size="sm"
                      value={llmBaseURL}
                      onChange={(e) => {
                        setLLMBaseURLState(e.target.value);
                        setLLMConfigSaved(false);
                      }}
                      placeholder="https://api.openai.com"
                      aria-label="LLM Base URL"
                    />
                  </div>
                )}

                <Button
                  variant="primary"
                  size="sm"
                  onClick={handleSaveAdvancedConfig}
                  loading={llmConfigSaving}
                >
                  {llmConfigSaved ? "Saved" : "Save advanced settings"}
                </Button>
              </div>
            </section>
          )}

          {(llmMode === "cloud" ||
            (llmMode === "advanced" && llmProvider === "openai")) && (
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
                  onClick={() =>
                    Browser.OpenURL("https://platform.openai.com/api-keys")
                  }
                >
                  Obtain an API Key
                </Button>
              )}
            </section>
          )}
        </div>
      </div>

      {shortcutsHelpOpen && (
        <ShortcutsHelp onClose={() => setShortcutsHelpOpen(false)} />
      )}
    </div>
  );
}
