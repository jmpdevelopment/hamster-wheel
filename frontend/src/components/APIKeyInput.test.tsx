import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { APIKeyInput } from "./APIKeyInput";

const mockGetReedAPIKey = vi.fn();
const mockSetReedAPIKey = vi.fn();

vi.mock("../../bindings/hamster-wheel/settingsservice", () => ({
  GetReedAPIKey: (...args: unknown[]) => mockGetReedAPIKey(...args),
  SetReedAPIKey: (...args: unknown[]) => mockSetReedAPIKey(...args),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mockGetReedAPIKey.mockResolvedValue("");
  mockSetReedAPIKey.mockResolvedValue(undefined);
});

describe("APIKeyInput", () => {
  it("renders the input and save button", async () => {
    render(<APIKeyInput onError={vi.fn()} />);

    expect(screen.getByLabelText("Reed API Key")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
  });

  it("shows hint when no key is set", async () => {
    render(<APIKeyInput onError={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText(/reed\.co\.uk\/developers/i)).toBeInTheDocument();
    });
  });

  it("hides hint when key is already saved", async () => {
    mockGetReedAPIKey.mockResolvedValue("existing-key");

    render(<APIKeyInput onError={vi.fn()} />);

    await waitFor(() => {
      expect(screen.queryByText(/reed\.co\.uk\/developers/i)).not.toBeInTheDocument();
    });
  });

  it("saves the key when Save is clicked", async () => {
    render(<APIKeyInput onError={vi.fn()} />);

    const input = screen.getByLabelText("Reed API Key");
    await userEvent.type(input, "my-api-key");
    await userEvent.click(screen.getByRole("button", { name: /save/i }));

    expect(mockSetReedAPIKey).toHaveBeenCalledWith("my-api-key");
  });

  it("shows Saved confirmation after saving", async () => {
    render(<APIKeyInput onError={vi.fn()} />);

    const input = screen.getByLabelText("Reed API Key");
    await userEvent.type(input, "my-api-key");
    await userEvent.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(screen.getByText("Saved")).toBeInTheDocument();
    });
  });

  it("calls onError when save fails", async () => {
    mockSetReedAPIKey.mockRejectedValue(new Error("db error"));
    const onError = vi.fn();

    render(<APIKeyInput onError={onError} />);

    const input = screen.getByLabelText("Reed API Key");
    await userEvent.type(input, "bad-key");
    await userEvent.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith("db error");
    });
  });

  it("disables Save button when input is empty", () => {
    render(<APIKeyInput onError={vi.fn()} />);

    const button = screen.getByRole("button", { name: /save/i });
    expect(button).toBeDisabled();
  });
});
