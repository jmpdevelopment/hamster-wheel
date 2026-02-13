import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { SettingsPanel } from "./SettingsPanel";

const mockGetReedAPIKey = vi.fn();
const mockSetReedAPIKey = vi.fn();
const mockOpenURL = vi.fn();

const mockSetKeyboardShortcuts = vi.fn();

vi.mock("../../bindings/hamster-wheel/settingsservice", () => ({
  GetReedAPIKey: (...args: unknown[]) => mockGetReedAPIKey(...args),
  SetReedAPIKey: (...args: unknown[]) => mockSetReedAPIKey(...args),
  SetKeyboardShortcuts: (...args: unknown[]) =>
    mockSetKeyboardShortcuts(...args),
}));

vi.mock("@wailsio/runtime", () => ({
  Browser: { OpenURL: (...args: unknown[]) => mockOpenURL(...args) },
}));

const defaultProps = {
  onClose: vi.fn(),
  theme: "system" as const,
  onSetTheme: vi.fn().mockResolvedValue(undefined),
  onError: vi.fn(),
  keyboardShortcuts: true,
  onSetKeyboardShortcuts: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
  mockGetReedAPIKey.mockResolvedValue("");
  mockSetReedAPIKey.mockResolvedValue(undefined);
  mockSetKeyboardShortcuts.mockResolvedValue(undefined);
  defaultProps.onClose = vi.fn();
  defaultProps.onSetTheme = vi.fn().mockResolvedValue(undefined);
  defaultProps.onError = vi.fn();
  defaultProps.onSetKeyboardShortcuts = vi.fn();
});

describe("SettingsPanel", () => {
  // --- Structure ---

  it("renders Settings title", () => {
    render(<SettingsPanel {...defaultProps} />);
    expect(screen.getByText("Settings")).toBeInTheDocument();
  });

  it("has dialog role with Settings label", () => {
    render(<SettingsPanel {...defaultProps} />);
    expect(
      screen.getByRole("dialog", { name: "Settings" })
    ).toBeInTheDocument();
  });

  it("calls onClose when close button is clicked", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await userEvent.click(
      screen.getByRole("button", { name: /close settings/i })
    );
    expect(defaultProps.onClose).toHaveBeenCalledOnce();
  });

  // --- API Key Section ---

  it("renders Reed API Key input", () => {
    render(<SettingsPanel {...defaultProps} />);
    expect(screen.getByLabelText("Reed API Key")).toBeInTheDocument();
  });

  it("shows Obtain a Key button when no key is set", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Obtain a Key" })
      ).toBeInTheDocument();
    });
  });

  it("hides Obtain a Key button when key exists", async () => {
    mockGetReedAPIKey.mockResolvedValue("existing-key");
    render(<SettingsPanel {...defaultProps} />);
    await waitFor(() => {
      expect(
        screen.queryByRole("button", { name: "Obtain a Key" })
      ).not.toBeInTheDocument();
    });
  });

  it("opens Reed developer page when Obtain a Key is clicked", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Obtain a Key" })
      ).toBeInTheDocument();
    });
    await userEvent.click(
      screen.getByRole("button", { name: "Obtain a Key" })
    );
    expect(mockOpenURL).toHaveBeenCalledWith(
      "https://www.reed.co.uk/developers/Jobseeker"
    );
  });

  it("disables Save when input is empty", () => {
    render(<SettingsPanel {...defaultProps} />);
    expect(screen.getByRole("button", { name: /save/i })).toBeDisabled();
  });

  it("saves API key when Save is clicked", async () => {
    render(<SettingsPanel {...defaultProps} />);
    const input = screen.getByLabelText("Reed API Key");
    await userEvent.type(input, "my-key");
    await userEvent.click(screen.getByRole("button", { name: /save/i }));
    expect(mockSetReedAPIKey).toHaveBeenCalledWith("my-key");
  });

  it("shows Saved confirmation after save", async () => {
    render(<SettingsPanel {...defaultProps} />);
    const input = screen.getByLabelText("Reed API Key");
    await userEvent.type(input, "my-key");
    await userEvent.click(screen.getByRole("button", { name: /save/i }));
    await waitFor(() => {
      expect(screen.getByText("Saved")).toBeInTheDocument();
    });
  });

  it("calls onError when save fails", async () => {
    mockSetReedAPIKey.mockRejectedValue(new Error("save failed"));
    render(<SettingsPanel {...defaultProps} />);
    const input = screen.getByLabelText("Reed API Key");
    await userEvent.type(input, "bad-key");
    await userEvent.click(screen.getByRole("button", { name: /save/i }));
    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith("save failed");
    });
  });

  // --- Theme Section ---

  it("renders theme buttons", () => {
    render(<SettingsPanel {...defaultProps} />);
    expect(screen.getByRole("button", { name: "System" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Dark" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Light" })).toBeInTheDocument();
  });

  it("marks the active theme button as pressed", () => {
    render(<SettingsPanel {...defaultProps} theme="dark" />);
    expect(screen.getByRole("button", { name: "Dark" })).toHaveAttribute(
      "aria-pressed",
      "true"
    );
    expect(screen.getByRole("button", { name: "Light" })).toHaveAttribute(
      "aria-pressed",
      "false"
    );
    expect(screen.getByRole("button", { name: "System" })).toHaveAttribute(
      "aria-pressed",
      "false"
    );
  });

  it("calls onSetTheme when a theme button is clicked", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await userEvent.click(screen.getByRole("button", { name: "Light" }));
    expect(defaultProps.onSetTheme).toHaveBeenCalledWith("light");
  });

  it("highlights system button when theme is system", () => {
    render(<SettingsPanel {...defaultProps} theme="system" />);
    expect(screen.getByRole("button", { name: "System" })).toHaveAttribute(
      "aria-pressed",
      "true"
    );
  });

  // --- Keyboard Shortcuts Section ---

  it("renders keyboard shortcuts toggle buttons", () => {
    render(<SettingsPanel {...defaultProps} />);
    expect(
      screen.getByRole("button", { name: "Enabled" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Disabled" })
    ).toBeInTheDocument();
  });

  it("marks Enabled as pressed when shortcuts are enabled", () => {
    render(<SettingsPanel {...defaultProps} keyboardShortcuts={true} />);
    expect(screen.getByRole("button", { name: "Enabled" })).toHaveAttribute(
      "aria-pressed",
      "true"
    );
    expect(screen.getByRole("button", { name: "Disabled" })).toHaveAttribute(
      "aria-pressed",
      "false"
    );
  });

  it("marks Disabled as pressed when shortcuts are disabled", () => {
    render(<SettingsPanel {...defaultProps} keyboardShortcuts={false} />);
    expect(screen.getByRole("button", { name: "Disabled" })).toHaveAttribute(
      "aria-pressed",
      "true"
    );
    expect(screen.getByRole("button", { name: "Enabled" })).toHaveAttribute(
      "aria-pressed",
      "false"
    );
  });

  it("calls SetKeyboardShortcuts and onSetKeyboardShortcuts when Disabled is clicked", async () => {
    render(<SettingsPanel {...defaultProps} keyboardShortcuts={true} />);
    await userEvent.click(screen.getByRole("button", { name: "Disabled" }));
    expect(mockSetKeyboardShortcuts).toHaveBeenCalledWith("false");
    await waitFor(() => {
      expect(defaultProps.onSetKeyboardShortcuts).toHaveBeenCalledWith(false);
    });
  });

  it("calls SetKeyboardShortcuts and onSetKeyboardShortcuts when Enabled is clicked", async () => {
    render(<SettingsPanel {...defaultProps} keyboardShortcuts={false} />);
    await userEvent.click(screen.getByRole("button", { name: "Enabled" }));
    expect(mockSetKeyboardShortcuts).toHaveBeenCalledWith("true");
    await waitFor(() => {
      expect(defaultProps.onSetKeyboardShortcuts).toHaveBeenCalledWith(true);
    });
  });

  it("calls onError when keyboard shortcuts save fails", async () => {
    mockSetKeyboardShortcuts.mockRejectedValue(new Error("save failed"));
    render(<SettingsPanel {...defaultProps} keyboardShortcuts={true} />);
    await userEvent.click(screen.getByRole("button", { name: "Disabled" }));
    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith("save failed");
    });
  });

  it("opens shortcuts help overlay when ? button is clicked", async () => {
    render(<SettingsPanel {...defaultProps} />);
    await userEvent.click(
      screen.getByRole("button", { name: /show keyboard shortcuts/i })
    );
    expect(
      screen.getByRole("dialog", { name: "Keyboard shortcuts" })
    ).toBeInTheDocument();
  });
});
