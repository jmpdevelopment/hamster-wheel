import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SettingsPanel } from "./SettingsPanel";

const mockHasReedAPIKey = vi.fn();
const mockSetReedAPIKey = vi.fn();
const mockClearReedAPIKey = vi.fn();
const mockHasAdzunaCredentials = vi.fn();
const mockSetAdzunaCredentials = vi.fn();
const mockClearAdzunaCredentials = vi.fn();
const mockHasOpenAIAPIKey = vi.fn();
const mockSetOpenAIAPIKey = vi.fn();
const mockClearOpenAIAPIKey = vi.fn();
const mockGetAutoPollingEnabled = vi.fn();
const mockSetAutoPollingEnabled = vi.fn();
const mockGetPollIntervalMinutes = vi.fn();
const mockSetPollIntervalMinutes = vi.fn();
const mockGetJobRetentionDays = vi.fn();
const mockSetJobRetentionDays = vi.fn();
const mockGetAutoMatchEnabled = vi.fn();
const mockSetAutoMatchEnabled = vi.fn();
const mockGetAutoMatchLimit = vi.fn();
const mockSetAutoMatchLimit = vi.fn();
const mockApplyPollingPaused = vi.fn();
const mockApplyPollingIntervalMinutes = vi.fn();
const mockGetLLMMode = vi.fn();
const mockSetLLMMode = vi.fn();
const mockGetLLMProvider = vi.fn();
const mockSetLLMProvider = vi.fn();
const mockGetLLMModel = vi.fn();
const mockSetLLMModel = vi.fn();
const mockGetLLMBaseURL = vi.fn();
const mockSetLLMBaseURL = vi.fn();
const mockGetLocalRuntimeStatus = vi.fn();
const mockGetLocalRuntimeModels = vi.fn();
const mockGetLocalRuntimeModel = vi.fn();
const mockGetLocalRuntimePullProgress = vi.fn();
const mockPullLocalRuntimeModel = vi.fn();
const mockStartLocalRuntime = vi.fn();
const mockStopLocalRuntime = vi.fn();
const mockSetLocalRuntimeEngine = vi.fn();
const mockSetLocalRuntimeModel = vi.fn();
const mockGetCVPath = vi.fn();
const mockSetCVPath = vi.fn();
const mockSetKeyboardShortcuts = vi.fn();
const mockOpenURL = vi.fn();
const mockOpenFile = vi.fn();

vi.mock("../../bindings/hamster-wheel/settingsservice", () => ({
  HasReedAPIKey: (...args: unknown[]) => mockHasReedAPIKey(...args),
  SetReedAPIKey: (...args: unknown[]) => mockSetReedAPIKey(...args),
  ClearReedAPIKey: (...args: unknown[]) => mockClearReedAPIKey(...args),
  HasAdzunaCredentials: (...args: unknown[]) =>
    mockHasAdzunaCredentials(...args),
  SetAdzunaCredentials: (...args: unknown[]) =>
    mockSetAdzunaCredentials(...args),
  ClearAdzunaCredentials: (...args: unknown[]) =>
    mockClearAdzunaCredentials(...args),
  HasOpenAIAPIKey: (...args: unknown[]) => mockHasOpenAIAPIKey(...args),
  SetOpenAIAPIKey: (...args: unknown[]) => mockSetOpenAIAPIKey(...args),
  ClearOpenAIAPIKey: (...args: unknown[]) => mockClearOpenAIAPIKey(...args),
  GetAutoPollingEnabled: (...args: unknown[]) =>
    mockGetAutoPollingEnabled(...args),
  SetAutoPollingEnabled: (...args: unknown[]) =>
    mockSetAutoPollingEnabled(...args),
  GetPollIntervalMinutes: (...args: unknown[]) =>
    mockGetPollIntervalMinutes(...args),
  SetPollIntervalMinutes: (...args: unknown[]) =>
    mockSetPollIntervalMinutes(...args),
  GetJobRetentionDays: (...args: unknown[]) =>
    mockGetJobRetentionDays(...args),
  SetJobRetentionDays: (...args: unknown[]) =>
    mockSetJobRetentionDays(...args),
  GetAutoMatchEnabled: (...args: unknown[]) => mockGetAutoMatchEnabled(...args),
  SetAutoMatchEnabled: (...args: unknown[]) => mockSetAutoMatchEnabled(...args),
  GetAutoMatchLimit: (...args: unknown[]) => mockGetAutoMatchLimit(...args),
  SetAutoMatchLimit: (...args: unknown[]) => mockSetAutoMatchLimit(...args),
  GetLLMMode: (...args: unknown[]) => mockGetLLMMode(...args),
  SetLLMMode: (...args: unknown[]) => mockSetLLMMode(...args),
  GetLLMProvider: (...args: unknown[]) => mockGetLLMProvider(...args),
  SetLLMProvider: (...args: unknown[]) => mockSetLLMProvider(...args),
  GetLLMModel: (...args: unknown[]) => mockGetLLMModel(...args),
  SetLLMModel: (...args: unknown[]) => mockSetLLMModel(...args),
  GetLLMBaseURL: (...args: unknown[]) => mockGetLLMBaseURL(...args),
  SetLLMBaseURL: (...args: unknown[]) => mockSetLLMBaseURL(...args),
  GetLocalRuntimeStatus: (...args: unknown[]) =>
    mockGetLocalRuntimeStatus(...args),
  GetLocalRuntimeModels: (...args: unknown[]) =>
    mockGetLocalRuntimeModels(...args),
  GetLocalRuntimeModel: (...args: unknown[]) => mockGetLocalRuntimeModel(...args),
  GetLocalRuntimePullProgress: (...args: unknown[]) =>
    mockGetLocalRuntimePullProgress(...args),
  PullLocalRuntimeModel: (...args: unknown[]) =>
    mockPullLocalRuntimeModel(...args),
  StartLocalRuntime: (...args: unknown[]) => mockStartLocalRuntime(...args),
  StopLocalRuntime: (...args: unknown[]) => mockStopLocalRuntime(...args),
  SetLocalRuntimeEngine: (...args: unknown[]) =>
    mockSetLocalRuntimeEngine(...args),
  SetLocalRuntimeModel: (...args: unknown[]) =>
    mockSetLocalRuntimeModel(...args),
  GetCVPath: (...args: unknown[]) => mockGetCVPath(...args),
  SetCVPath: (...args: unknown[]) => mockSetCVPath(...args),
  SetKeyboardShortcuts: (...args: unknown[]) =>
    mockSetKeyboardShortcuts(...args),
}));

vi.mock("../../bindings/hamster-wheel/pollingservice", () => ({
  SetPollingPaused: (...args: unknown[]) => mockApplyPollingPaused(...args),
  SetPollingIntervalMinutes: (...args: unknown[]) =>
    mockApplyPollingIntervalMinutes(...args),
}));

vi.mock("@wailsio/runtime", () => ({
  Browser: { OpenURL: (...args: unknown[]) => mockOpenURL(...args) },
  Dialogs: {
    SaveFile: vi.fn().mockResolvedValue(""),
    OpenFile: (...args: unknown[]) => mockOpenFile(...args),
  },
}));

const defaultProps = {
  onClose: vi.fn(),
  theme: "system" as const,
  onSetTheme: vi.fn().mockResolvedValue(undefined),
  onError: vi.fn(),
  keyboardShortcuts: true,
  onSetKeyboardShortcuts: vi.fn(),
};

const openTab = async (name: string) => {
  await userEvent.click(screen.getByRole("tab", { name }));
};

beforeEach(() => {
  vi.clearAllMocks();
  mockHasReedAPIKey.mockResolvedValue(false);
  mockSetReedAPIKey.mockResolvedValue(undefined);
  mockClearReedAPIKey.mockResolvedValue(undefined);
  mockHasAdzunaCredentials.mockResolvedValue(false);
  mockSetAdzunaCredentials.mockResolvedValue(undefined);
  mockClearAdzunaCredentials.mockResolvedValue(undefined);
  mockHasOpenAIAPIKey.mockResolvedValue(false);
  mockSetOpenAIAPIKey.mockResolvedValue(undefined);
  mockClearOpenAIAPIKey.mockResolvedValue(undefined);
  mockGetAutoPollingEnabled.mockResolvedValue(false);
  mockSetAutoPollingEnabled.mockResolvedValue(undefined);
  mockGetPollIntervalMinutes.mockResolvedValue(30);
  mockSetPollIntervalMinutes.mockResolvedValue(undefined);
  mockGetJobRetentionDays.mockResolvedValue(30);
  mockSetJobRetentionDays.mockResolvedValue(undefined);
  mockGetAutoMatchEnabled.mockResolvedValue(true);
  mockSetAutoMatchEnabled.mockResolvedValue(undefined);
  mockGetAutoMatchLimit.mockResolvedValue(0);
  mockSetAutoMatchLimit.mockResolvedValue(undefined);
  mockApplyPollingPaused.mockResolvedValue(undefined);
  mockApplyPollingIntervalMinutes.mockResolvedValue(undefined);
  mockGetLLMMode.mockResolvedValue("cloud");
  mockSetLLMMode.mockResolvedValue(undefined);
  mockGetLLMProvider.mockResolvedValue("openai");
  mockSetLLMProvider.mockResolvedValue(undefined);
  mockGetLLMModel.mockResolvedValue("gpt-4o-mini");
  mockSetLLMModel.mockResolvedValue(undefined);
  mockGetLLMBaseURL.mockResolvedValue("");
  mockSetLLMBaseURL.mockResolvedValue(undefined);
  mockGetLocalRuntimeStatus.mockResolvedValue({
    status: "ready",
    message: "",
    startedByApp: false,
  });
  mockGetLocalRuntimeModels.mockResolvedValue({
    installed: [{ name: "llama3.1:8b" }],
  });
  mockGetLocalRuntimeModel.mockResolvedValue("llama3.1:8b");
  mockGetLocalRuntimePullProgress.mockResolvedValue({
    active: false,
    model: "llama3.1:8b",
    status: "",
    message: "",
    totalBytes: 0,
    completedBytes: 0,
    percent: 0,
    ready: false,
  });
  mockPullLocalRuntimeModel.mockResolvedValue({
    model: "llama3.1:8b",
    ready: true,
    status: "success",
  });
  mockStartLocalRuntime.mockResolvedValue({
    status: "ready",
    message: "",
    startedByApp: true,
  });
  mockStopLocalRuntime.mockResolvedValue({
    status: "stopped",
    message: "",
    startedByApp: false,
  });
  mockSetLocalRuntimeEngine.mockResolvedValue(undefined);
  mockSetLocalRuntimeModel.mockResolvedValue(undefined);
  mockGetCVPath.mockResolvedValue("");
  mockSetCVPath.mockResolvedValue(undefined);
  mockSetKeyboardShortcuts.mockResolvedValue(undefined);
  mockOpenFile.mockResolvedValue("");
  defaultProps.onClose = vi.fn();
  defaultProps.onSetTheme = vi.fn().mockResolvedValue(undefined);
  defaultProps.onError = vi.fn();
  defaultProps.onSetKeyboardShortcuts = vi.fn();
});

describe("SettingsPanel", () => {
  it("renders dialog title and tab controls", () => {
    render(<SettingsPanel {...defaultProps} />);

    expect(
      screen.getByRole("dialog", { name: "Settings" })
    ).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Interface" })).toBeInTheDocument();
    expect(
      screen.getByRole("tab", { name: "Jobs Providers" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("tab", { name: "LLM Providers" })
    ).toBeInTheDocument();
  });

  it("shows interface settings by default", () => {
    render(<SettingsPanel {...defaultProps} />);

    expect(screen.getByText("Theme")).toBeInTheDocument();
    expect(screen.getByText("Keyboard Shortcuts")).toBeInTheDocument();
    expect(
      document.getElementById("settings-panel-jobs-providers")
    ).toHaveAttribute("hidden");
    expect(
      document.getElementById("settings-panel-llm-providers")
    ).toHaveAttribute("hidden");
  });

  it("switches tabs between interface, jobs providers, and llm providers", async () => {
    render(<SettingsPanel {...defaultProps} />);

    await openTab("Jobs Providers");
    expect(await screen.findByLabelText("Reed API Key")).toBeInTheDocument();

    await openTab("LLM Providers");
    expect(await screen.findByText("Matching Mode")).toBeInTheDocument();
    expect(screen.getByLabelText("LLM Mode")).toBeInTheDocument();
    expect(screen.getByLabelText("Cloud LLM Model")).toBeInTheDocument();
    expect(screen.queryByLabelText("LLM Base URL")).not.toBeInTheDocument();

    await openTab("Interface");
    expect(screen.getByText("Theme")).toBeInTheDocument();
  });

  it("calls onClose when close button is clicked", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await userEvent.click(
      screen.getByRole("button", { name: /close settings/i })
    );
    expect(defaultProps.onClose).toHaveBeenCalledOnce();
  });

  it("persists theme choice from interface tab", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await userEvent.click(screen.getByRole("button", { name: "Light" }));
    expect(defaultProps.onSetTheme).toHaveBeenCalledWith("light");
  });

  it("reports theme save errors", async () => {
    defaultProps.onSetTheme = vi.fn().mockRejectedValue(new Error("theme failed"));
    render(<SettingsPanel {...defaultProps} />);

    await userEvent.click(screen.getByRole("button", { name: "Light" }));
    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith("theme failed");
    });
  });

  it("persists keyboard shortcuts toggle from interface tab", async () => {
    render(<SettingsPanel {...defaultProps} keyboardShortcuts={true} />);
    await userEvent.click(screen.getByRole("button", { name: "Disabled" }));

    expect(mockSetKeyboardShortcuts).toHaveBeenCalledWith("false");
    await waitFor(() => {
      expect(defaultProps.onSetKeyboardShortcuts).toHaveBeenCalledWith(false);
    });
  });

  it("reports keyboard shortcuts errors", async () => {
    mockSetKeyboardShortcuts.mockRejectedValue(new Error("save failed"));
    render(<SettingsPanel {...defaultProps} keyboardShortcuts={true} />);
    await userEvent.click(screen.getByRole("button", { name: "Disabled" }));
    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith("save failed");
    });
  });

  it("opens shortcuts help overlay from interface tab", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await userEvent.click(
      screen.getByRole("button", { name: /show keyboard shortcuts/i })
    );
    expect(
      screen.getByRole("dialog", { name: "Keyboard shortcuts" })
    ).toBeInTheDocument();
  });

  it("shows Reed obtain button when key is missing", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    expect(
      await screen.findByRole("button", { name: "Obtain a Key" })
    ).toBeInTheDocument();
  });

  it("shows Reed clear controls when key exists", async () => {
    mockHasReedAPIKey.mockResolvedValue(true);
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    expect(await screen.findByRole("button", { name: "Clear" })).toBeInTheDocument();
    expect(
      screen.getByText("Key is stored securely in your OS keychain.")
    ).toBeInTheDocument();
  });

  it("saves and clears Reed API key", async () => {
    mockHasReedAPIKey.mockResolvedValue(true);
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    await userEvent.type(screen.getByLabelText("Reed API Key"), "reed-key");
    await userEvent.click(screen.getByRole("button", { name: /^Save$/ }));
    expect(mockSetReedAPIKey).toHaveBeenCalledWith("reed-key");

    await userEvent.click(await screen.findByRole("button", { name: "Clear" }));
    expect(mockClearReedAPIKey).toHaveBeenCalledOnce();
  });

  it("opens Reed developer URL", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    await userEvent.click(screen.getByRole("button", { name: "Obtain a Key" }));
    expect(mockOpenURL).toHaveBeenCalledWith(
      "https://www.reed.co.uk/developers/Jobseeker"
    );
  });

  it("shows Adzuna obtain button when credentials are missing", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    expect(
      await screen.findByRole("button", { name: "Obtain Credentials" })
    ).toBeInTheDocument();
  });

  it("shows Adzuna clear controls when credentials exist", async () => {
    mockHasAdzunaCredentials.mockResolvedValue(true);
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    expect(
      await screen.findByRole("button", { name: "Clear Credentials" })
    ).toBeInTheDocument();
    expect(
      screen.getByText("Credentials are stored securely in your OS keychain.")
    ).toBeInTheDocument();
  });

  it("saves and clears Adzuna credentials", async () => {
    mockHasAdzunaCredentials.mockResolvedValue(true);
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    await userEvent.type(screen.getByLabelText("Adzuna App ID"), "adzuna-id");
    await userEvent.type(screen.getByLabelText("Adzuna App Key"), "adzuna-key");
    await userEvent.click(screen.getByRole("button", { name: "Save Credentials" }));
    expect(mockSetAdzunaCredentials).toHaveBeenCalledWith("adzuna-id", "adzuna-key");

    await userEvent.click(
      await screen.findByRole("button", { name: "Clear Credentials" })
    );
    expect(mockClearAdzunaCredentials).toHaveBeenCalledOnce();
  });

  it("opens Adzuna developer URL", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    await userEvent.click(screen.getByRole("button", { name: "Obtain Credentials" }));
    expect(mockOpenURL).toHaveBeenCalledWith("https://developer.adzuna.com");
  });

  it("loads and saves auto polling settings", async () => {
    mockGetAutoPollingEnabled.mockResolvedValue(true);
    mockGetPollIntervalMinutes.mockResolvedValue(90);
    mockGetJobRetentionDays.mockResolvedValue(21);

    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Enabled" })).toHaveAttribute(
        "aria-pressed",
        "true"
      );
      expect(screen.getByLabelText("Poll Interval Minutes")).toHaveValue(90);
      expect(screen.getByLabelText("Job Retention Days")).toHaveValue(21);
    });

    await userEvent.click(screen.getByRole("button", { name: "Disabled" }));
    await userEvent.clear(screen.getByLabelText("Poll Interval Minutes"));
    await userEvent.type(screen.getByLabelText("Poll Interval Minutes"), "60");
    await userEvent.clear(screen.getByLabelText("Job Retention Days"));
    await userEvent.type(screen.getByLabelText("Job Retention Days"), "14");
    await userEvent.click(
      screen.getByRole("button", { name: "Save polling settings" })
    );

    expect(mockSetAutoPollingEnabled).toHaveBeenCalledWith(false);
    expect(mockSetPollIntervalMinutes).toHaveBeenCalledWith(60);
    expect(mockSetJobRetentionDays).toHaveBeenCalledWith(14);
    expect(mockApplyPollingIntervalMinutes).toHaveBeenCalledWith(60);
    expect(mockApplyPollingPaused).toHaveBeenCalledWith(true);
  });

  it("reports validation errors for invalid poll interval", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    await userEvent.clear(screen.getByLabelText("Poll Interval Minutes"));
    await userEvent.type(screen.getByLabelText("Poll Interval Minutes"), "5");
    await userEvent.click(
      screen.getByRole("button", { name: "Save polling settings" })
    );

    expect(defaultProps.onError).toHaveBeenCalledWith(
      "Polling interval must be a whole number between 30 and 1440 minutes."
    );
    expect(mockSetPollIntervalMinutes).not.toHaveBeenCalled();
  });

  it("reports validation errors for invalid job retention days", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("Jobs Providers");

    await userEvent.clear(screen.getByLabelText("Job Retention Days"));
    await userEvent.type(screen.getByLabelText("Job Retention Days"), "31");
    await userEvent.click(
      screen.getByRole("button", { name: "Save polling settings" })
    );

    expect(defaultProps.onError).toHaveBeenCalledWith(
      "Job retention days must be a whole number between 1 and 30."
    );
    expect(mockSetJobRetentionDays).not.toHaveBeenCalled();
  });

  it("loads cloud mode by default and hides advanced endpoint fields", async () => {
    mockHasOpenAIAPIKey.mockResolvedValue(true);
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await waitFor(() => {
      expect(screen.getByRole("radio", { name: "Cloud" })).toHaveAttribute(
        "aria-checked",
        "true"
      );
      expect(screen.getByLabelText("Cloud LLM Model")).toHaveValue("gpt-4o-mini");
    });
    expect(screen.queryByLabelText("LLM Base URL")).not.toBeInTheDocument();
    expect(
      screen.getByText("Key is stored securely in your OS keychain.")
    ).toBeInTheDocument();
  });

  it("loads and saves auto match settings", async () => {
    mockGetAutoMatchEnabled.mockResolvedValue(false);
    mockGetAutoMatchLimit.mockResolvedValue(12);

    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Disabled" })).toHaveAttribute(
        "aria-pressed",
        "true"
      );
      expect(screen.getByLabelText("Auto Match Limit")).toHaveValue(12);
    });

    await userEvent.click(screen.getByRole("button", { name: "Enabled" }));
    await userEvent.clear(screen.getByLabelText("Auto Match Limit"));
    await userEvent.type(screen.getByLabelText("Auto Match Limit"), "5");
    await userEvent.click(
      screen.getByRole("button", { name: "Save auto-match settings" })
    );

    expect(mockSetAutoMatchEnabled).toHaveBeenCalledWith(true);
    expect(mockSetAutoMatchLimit).toHaveBeenCalledWith(5);
  });

  it("reports validation errors for invalid auto match limit", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.clear(screen.getByLabelText("Auto Match Limit"));
    await userEvent.type(screen.getByLabelText("Auto Match Limit"), "-1");
    await userEvent.click(
      screen.getByRole("button", { name: "Save auto-match settings" })
    );

    expect(defaultProps.onError).toHaveBeenCalledWith(
      "Auto-match limit must be a whole number greater than or equal to 0."
    );
    expect(mockSetAutoMatchLimit).not.toHaveBeenCalled();
  });

  it("switches mode sections and gates base url to advanced only", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.click(screen.getByRole("radio", { name: "Local" }));
    expect(screen.getByLabelText("Local Runtime Model")).toBeInTheDocument();
    expect(screen.queryByLabelText("OpenAI API Key")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("LLM Base URL")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("radio", { name: "Advanced" }));
    expect(screen.getByLabelText("LLM Provider")).toBeInTheDocument();
    expect(screen.getByLabelText("LLM Model")).toBeInTheDocument();
    expect(screen.getByLabelText("LLM Base URL")).toBeInTheDocument();
    expect(screen.getByLabelText("OpenAI API Key")).toBeInTheDocument();

    await userEvent.selectOptions(screen.getByLabelText("LLM Provider"), "heuristic_v1");
    expect(screen.queryByLabelText("LLM Base URL")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("OpenAI API Key")).not.toBeInTheDocument();
  });

  it("saves cloud mode settings", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.selectOptions(screen.getByLabelText("Cloud LLM Model"), "gpt-4o");
    await userEvent.click(
      screen.getByRole("button", { name: "Save cloud settings" })
    );

    expect(mockSetLLMMode).toHaveBeenCalledWith("cloud");
    expect(mockSetLLMProvider).toHaveBeenCalledWith("openai");
    expect(mockSetLLMModel).toHaveBeenCalledWith("gpt-4o");
    expect(mockSetLLMBaseURL).toHaveBeenCalledWith("");
  });

  it("saves local mode settings", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.click(screen.getByRole("radio", { name: "Local" }));
    await userEvent.click(
      screen.getByRole("button", { name: "Save local settings" })
    );

    expect(mockSetLLMMode).toHaveBeenCalledWith("local");
    expect(mockSetLocalRuntimeEngine).toHaveBeenCalledWith("ollama");
    expect(mockSetLocalRuntimeModel).toHaveBeenCalledWith("llama3.1:8b");
  });

  it("starts local runtime from explicit control", async () => {
    mockGetLocalRuntimeStatus.mockResolvedValue({
      status: "stopped",
      message: "",
      startedByApp: false,
    });
    mockGetLocalRuntimeModels.mockResolvedValue({
      installed: [],
    });

    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");
    await userEvent.click(screen.getByRole("radio", { name: "Local" }));

    await userEvent.click(screen.getByRole("button", { name: "Start runtime" }));

    expect(mockStartLocalRuntime).toHaveBeenCalled();
    expect(mockPullLocalRuntimeModel).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Run guided setup" })).not.toBeInTheDocument();
  });

  it("downloads llama from explicit control", async () => {
    mockGetLocalRuntimeStatus.mockResolvedValue({
      status: "ready",
      message: "",
      startedByApp: true,
    });
    mockGetLocalRuntimeModels.mockResolvedValue({
      installed: [],
    });

    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");
    await userEvent.click(screen.getByRole("radio", { name: "Local" }));

    await userEvent.click(screen.getByRole("button", { name: "Download Llama" }));

    expect(mockPullLocalRuntimeModel).toHaveBeenCalledWith("llama3.1:8b");
    expect(screen.queryByRole("button", { name: "Run guided setup" })).not.toBeInTheDocument();
  });

  it("shows in-progress llama download state and disables duplicate download action", async () => {
    mockGetLocalRuntimeStatus.mockResolvedValue({
      status: "ready",
      message: "",
      startedByApp: true,
    });
    mockGetLocalRuntimeModels.mockResolvedValue({
      installed: [],
    });
    mockGetLocalRuntimePullProgress.mockResolvedValue({
      active: true,
      model: "llama3.1:8b",
      status: "downloading",
      message: "downloading",
      totalBytes: 1024,
      completedBytes: 512,
      percent: 50,
      ready: false,
    });

    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");
    await userEvent.click(screen.getByRole("radio", { name: "Local" }));

    expect(screen.getByText(/Download status:/)).toBeInTheDocument();
    expect(screen.getByText("50.0% (512 B / 1.0 KB)")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Llama download progress" })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Downloading Llama..." })
    ).toBeDisabled();
  });

  it("hides download status panel after completed pull", async () => {
    mockGetLocalRuntimeStatus.mockResolvedValue({
      status: "ready",
      message: "",
      startedByApp: true,
    });
    mockGetLocalRuntimeModels.mockResolvedValue({
      installed: [{ name: "llama3.1:8b" }],
    });
    mockGetLocalRuntimePullProgress.mockResolvedValue({
      active: false,
      model: "llama3.1:8b",
      status: "completed",
      message: "completed",
      totalBytes: 1024,
      completedBytes: 1024,
      percent: 100,
      ready: true,
    });

    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");
    await userEvent.click(screen.getByRole("radio", { name: "Local" }));

    expect(screen.queryByText(/Download status:/)).not.toBeInTheDocument();
  });

  it("shows local runtime system requirements guidance", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");
    await userEvent.click(screen.getByRole("radio", { name: "Local" }));

    expect(
      screen.getByText("System requirements (Llama 3.1 8B local mode)")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Impact while running: higher CPU\/RAM usage/i)
    ).toBeInTheDocument();
  });

  it("shows install action when ollama is missing", async () => {
    mockGetLocalRuntimeStatus.mockResolvedValue({
      status: "not_installed",
      message: "Install Ollama, open it once, then return to local model setup.",
      startedByApp: false,
    });

    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");
    await userEvent.click(screen.getByRole("radio", { name: "Local" }));

    await userEvent.click(screen.getByRole("button", { name: "Install Ollama" }));
    expect(mockOpenURL).toHaveBeenCalledWith("https://ollama.com/download");
  });

  it("saves advanced mode settings", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.click(screen.getByRole("radio", { name: "Advanced" }));
    await userEvent.selectOptions(screen.getByLabelText("LLM Provider"), "openai");
    await userEvent.clear(screen.getByLabelText("LLM Model"));
    await userEvent.type(screen.getByLabelText("LLM Model"), "custom-model");
    await userEvent.type(
      screen.getByLabelText("LLM Base URL"),
      "https://gateway.example/v1"
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Save advanced settings" })
    );

    expect(mockSetLLMMode).toHaveBeenCalledWith("advanced");
    expect(mockSetLLMProvider).toHaveBeenCalledWith("openai");
    expect(mockSetLLMModel).toHaveBeenCalledWith("custom-model");
    expect(mockSetLLMBaseURL).toHaveBeenCalledWith("https://gateway.example/v1");
  });

  it("loads, browses, saves, and clears CV path", async () => {
    mockGetCVPath.mockResolvedValue("/tmp/cv.txt");
    mockOpenFile.mockResolvedValue("/tmp/next-cv.txt");
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await waitFor(() => {
      expect(screen.getByLabelText("CV File Path")).toHaveValue("/tmp/cv.txt");
    });

    await userEvent.click(screen.getByRole("button", { name: "Browse" }));
    expect(screen.getByLabelText("CV File Path")).toHaveValue("/tmp/next-cv.txt");

    await userEvent.click(screen.getByRole("button", { name: "Save CV path" }));
    expect(mockSetCVPath).toHaveBeenCalledWith("/tmp/next-cv.txt");

    await userEvent.click(screen.getByRole("button", { name: "Clear" }));
    expect(mockSetCVPath).toHaveBeenCalledWith("");
  });

  it("reports llm configuration save errors", async () => {
    mockSetLLMModel.mockRejectedValue(new Error("model save failed"));
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.click(screen.getByRole("button", { name: "Save cloud settings" }));

    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith("model save failed");
    });
  });

  it("saves and clears OpenAI API key", async () => {
    mockHasOpenAIAPIKey.mockResolvedValue(true);
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.type(screen.getByLabelText("OpenAI API Key"), "sk-openai");
    await userEvent.click(screen.getByRole("button", { name: /^Save$/ }));
    expect(mockSetOpenAIAPIKey).toHaveBeenCalledWith("sk-openai");

    await userEvent.click(await screen.findByRole("button", { name: "Clear" }));
    expect(mockClearOpenAIAPIKey).toHaveBeenCalledOnce();
  });

  it("opens OpenAI key URL when key is missing", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.click(
      await screen.findByRole("button", { name: "Obtain an API Key" })
    );
    expect(mockOpenURL).toHaveBeenCalledWith("https://platform.openai.com/api-keys");
  });

  it("reports load errors from provider settings", async () => {
    mockGetLLMProvider.mockRejectedValue(new Error("provider load failed"));
    render(<SettingsPanel {...defaultProps} />);

    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith("provider load failed");
    });
  });

  it("reports CV path load errors", async () => {
    mockGetCVPath.mockRejectedValue(new Error("cv load failed"));
    render(<SettingsPanel {...defaultProps} />);

    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith("cv load failed");
    });
  });

  it("reports OpenAI API key save/clear errors", async () => {
    mockHasOpenAIAPIKey.mockResolvedValue(true);
    mockSetOpenAIAPIKey.mockRejectedValue(new Error("save key failed"));
    mockClearOpenAIAPIKey.mockRejectedValue(new Error("clear key failed"));

    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.type(screen.getByLabelText("OpenAI API Key"), "bad-key");
    await userEvent.click(screen.getByRole("button", { name: /^Save$/ }));
    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith("save key failed");
    });

    await userEvent.click(await screen.findByRole("button", { name: "Clear" }));
    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith("clear key failed");
    });
  });
});
