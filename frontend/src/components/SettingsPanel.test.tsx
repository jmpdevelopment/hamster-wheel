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
const mockGetLLMProvider = vi.fn();
const mockSetLLMProvider = vi.fn();
const mockGetLLMModel = vi.fn();
const mockSetLLMModel = vi.fn();
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
  GetLLMProvider: (...args: unknown[]) => mockGetLLMProvider(...args),
  SetLLMProvider: (...args: unknown[]) => mockSetLLMProvider(...args),
  GetLLMModel: (...args: unknown[]) => mockGetLLMModel(...args),
  SetLLMModel: (...args: unknown[]) => mockSetLLMModel(...args),
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
  mockGetLLMProvider.mockResolvedValue("openai");
  mockSetLLMProvider.mockResolvedValue(undefined);
  mockGetLLMModel.mockResolvedValue("gpt-4o-mini");
  mockSetLLMModel.mockResolvedValue(undefined);
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
    expect(await screen.findByLabelText("OpenAI API Key")).toBeInTheDocument();
    expect(screen.getByLabelText("LLM Provider")).toBeInTheDocument();

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

  it("loads and displays llm provider values", async () => {
    mockHasOpenAIAPIKey.mockResolvedValue(true);
    mockGetLLMProvider.mockResolvedValue("heuristic_v1");
    mockGetLLMModel.mockResolvedValue("heuristic_v1");

    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await waitFor(() => {
      expect(screen.getByLabelText("LLM Provider")).toHaveValue("heuristic_v1");
      expect(screen.getByLabelText("LLM Model")).toHaveValue("heuristic_v1");
    });
    expect(
      screen.getByText("Key is stored securely in your OS keychain.")
    ).toBeInTheDocument();
  });

  it("shows provider-specific model options", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    expect(screen.getByLabelText("LLM Model")).toHaveValue("gpt-4o-mini");
    expect(screen.getByRole("option", { name: "gpt-4o-mini" })).toBeInTheDocument();

    await userEvent.selectOptions(screen.getByLabelText("LLM Provider"), "heuristic_v1");
    await waitFor(() => {
      expect(screen.getByLabelText("LLM Model")).toHaveValue("heuristic_v1");
    });
    expect(
      screen.queryByRole("option", { name: "gpt-4o-mini" })
    ).not.toBeInTheDocument();
    expect(screen.getByRole("option", { name: "heuristic_v1" })).toBeInTheDocument();
  });

  it("normalizes incompatible saved model for the active provider", async () => {
    mockGetLLMProvider.mockResolvedValue("heuristic_v1");
    mockGetLLMModel.mockResolvedValue("gpt-4o");

    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await waitFor(() => {
      expect(screen.getByLabelText("LLM Provider")).toHaveValue("heuristic_v1");
      expect(screen.getByLabelText("LLM Model")).toHaveValue("heuristic_v1");
    });
  });

  it("saves llm provider/model settings", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await openTab("LLM Providers");

    await userEvent.selectOptions(screen.getByLabelText("LLM Provider"), "openai");
    await userEvent.selectOptions(screen.getByLabelText("LLM Model"), "gpt-4o");

    await userEvent.click(
      screen.getByRole("button", { name: "Save configuration" })
    );

    expect(mockSetLLMProvider).toHaveBeenCalledWith("openai");
    expect(mockSetLLMModel).toHaveBeenCalledWith("gpt-4o");
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

    await userEvent.click(
      screen.getByRole("button", { name: "Save configuration" })
    );

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
