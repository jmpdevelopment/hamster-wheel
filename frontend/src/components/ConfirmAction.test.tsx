import { render, screen, fireEvent, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, afterEach } from "vitest";
import { ConfirmAction } from "./ConfirmAction";

afterEach(() => {
  vi.useRealTimers();
});

describe("ConfirmAction", () => {
  it("renders trigger child", () => {
    render(
      <ConfirmAction onConfirm={() => {}}>
        <button>Delete</button>
      </ConfirmAction>
    );
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
  });

  it("shows confirm and cancel buttons when trigger is clicked", async () => {
    render(
      <ConfirmAction onConfirm={() => {}}>
        <button>Delete</button>
      </ConfirmAction>
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
  });

  it("uses custom confirmLabel", async () => {
    render(
      <ConfirmAction onConfirm={() => {}} confirmLabel="Delete Forever">
        <button>Delete</button>
      </ConfirmAction>
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(screen.getByRole("button", { name: "Delete Forever" })).toBeInTheDocument();
  });

  it("calls onConfirm when confirm button is clicked", async () => {
    const onConfirm = vi.fn();
    render(
      <ConfirmAction onConfirm={onConfirm}>
        <button>Delete</button>
      </ConfirmAction>
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    await userEvent.click(screen.getByRole("button", { name: "Confirm" }));
    expect(onConfirm).toHaveBeenCalledOnce();
  });

  it("does not call onConfirm when cancel is clicked", async () => {
    const onConfirm = vi.fn();
    render(
      <ConfirmAction onConfirm={onConfirm}>
        <button>Delete</button>
      </ConfirmAction>
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("resets to trigger after cancel", async () => {
    render(
      <ConfirmAction onConfirm={() => {}}>
        <button>Delete</button>
      </ConfirmAction>
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument();
  });

  it("resets state after confirm is clicked", async () => {
    render(
      <ConfirmAction onConfirm={() => {}}>
        <button>Delete</button>
      </ConfirmAction>
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    await userEvent.click(screen.getByRole("button", { name: "Confirm" }));
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
  });

  it("auto-resets after timeout", () => {
    vi.useFakeTimers();

    render(
      <ConfirmAction onConfirm={() => {}}>
        <button>Delete</button>
      </ConfirmAction>
    );

    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(3000);
    });
    expect(screen.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
  });

  it("does not auto-reset before timeout", () => {
    vi.useFakeTimers();

    render(
      <ConfirmAction onConfirm={() => {}}>
        <button>Delete</button>
      </ConfirmAction>
    );

    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    act(() => {
      vi.advanceTimersByTime(2999);
    });
    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();
  });

  it("resets on outside click", async () => {
    render(
      <div>
        <button data-testid="outside">Outside</button>
        <ConfirmAction onConfirm={() => {}}>
          <button>Delete</button>
        </ConfirmAction>
      </div>
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();

    await userEvent.click(screen.getByTestId("outside"));
    expect(screen.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
  });

  it("confirm button has danger variant classes", async () => {
    render(
      <ConfirmAction onConfirm={() => {}}>
        <button>Delete</button>
      </ConfirmAction>
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(screen.getByRole("button", { name: "Confirm" }).className).toContain(
      "bg-hw-danger"
    );
  });

  it("cancel button has secondary variant classes", async () => {
    render(
      <ConfirmAction onConfirm={() => {}}>
        <button>Delete</button>
      </ConfirmAction>
    );

    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(screen.getByRole("button", { name: "Cancel" }).className).toContain(
      "border-hw-border"
    );
  });
});
