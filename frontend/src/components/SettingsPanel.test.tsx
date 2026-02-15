import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SettingsPanel } from "./SettingsPanel";

const mockHasReedAPIKey = vi.fn();
const mockSetReedAPIKey = vi.fn();
const mockClearReedAPIKey = vi.fn();
const mockHasOpenAIAPIKey = vi.fn();
const mockSetOpenAIAPIKey = vi.fn();
const mockClearOpenAIAPIKey = vi.fn();
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
  HasOpenAIAPIKey: (...args: unknown[]) => mockHasOpenAIAPIKey(...args),
  SetOpenAIAPIKey: (...args: unknown[]) => mockSetOpenAIAPIKey(...args),
  ClearOpenAIAPIKey: (...args: unknown[]) => mockClearOpenAIAPIKey(...args),
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
  mockHasOpenAIAPIKey.mockResolvedValue(false);
  mockSetOpenAIAPIKey.mockResolvedValue(undefined);
  mockClearOpenAIAPIKey.mockResolvedValue(undefined);
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

  it("runs guided local setup by starting runtime and downloading llama", async () => {
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

    await userEvent.click(screen.getByRole("button", { name: "Run guided setup" }));

    expect(mockStartLocalRuntime).toHaveBeenCalled();
    expect(mockPullLocalRuntimeModel).toHaveBeenCalledWith("llama3.1:8b");
  });

  it("shows install action when ollama is missing", async () => {
    mockGetLocalRuntimeStatus.mockResolvedValue({
      status: "not_installed",
      message: "Install Ollama to enable guided local model mode.",
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
