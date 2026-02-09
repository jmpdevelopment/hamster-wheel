import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { JobDetail } from "./JobDetail";

// Mock Wails runtime.
vi.mock("@wailsio/runtime", () => ({
  Browser: { OpenURL: vi.fn() },
}));

import { Browser } from "@wailsio/runtime";

const fakeJob = (overrides = {}) => ({
  ID: "j1",
  Source: "reed_uk",
  SourceID: "src-j1",
  Title: "Senior Go Developer",
  Company: "Acme Corp",
  Location: "London",
  Description: "Build backend services in Go.\nWork with a great team.",
  URL: "https://example.com/job",
  PostedAt: "2026-02-08T10:00:00Z",
  DiscoveredAt: "2026-02-08T11:00:00Z",
  FilterID: "f1",
  ...overrides,
});

describe("JobDetail", () => {
  it("renders job title and details", () => {
    render(<JobDetail job={fakeJob()} onDelete={() => {}} onClose={() => {}} />);

    expect(screen.getByText("Senior Go Developer")).toBeInTheDocument();
    expect(screen.getByText(/Acme Corp/)).toBeInTheDocument();
    expect(screen.getByText(/London/)).toBeInTheDocument();
  });

  it("renders plain text description", () => {
    render(<JobDetail job={fakeJob()} onDelete={() => {}} onClose={() => {}} />);
    expect(screen.getByText(/Build backend services in Go/)).toBeInTheDocument();
  });

  it("renders HTML description as formatted content", () => {
    render(
      <JobDetail
        job={fakeJob({
          Description: "<p>Looking for a <strong>Go Developer</strong></p>",
        })}
        onDelete={() => {}}
        onClose={() => {}}
      />
    );
    expect(screen.getByText(/Looking for a/)).toBeInTheDocument();
    expect(screen.getByText("Go Developer")).toBeInTheDocument();
  });

  it("shows 'No description available' when description is empty", () => {
    render(
      <JobDetail
        job={fakeJob({ Description: "" })}
        onDelete={() => {}}
        onClose={() => {}}
      />
    );
    expect(screen.getByText("No description available.")).toBeInTheDocument();
  });

  it("calls BrowserOpenURL when Open in Browser is clicked", async () => {
    render(<JobDetail job={fakeJob()} onDelete={() => {}} onClose={() => {}} />);

    await userEvent.click(
      screen.getByRole("button", { name: /open in browser/i })
    );
    expect(Browser.OpenURL).toHaveBeenCalledWith("https://example.com/job");
  });

  it("hides Open in Browser when URL is empty", () => {
    render(
      <JobDetail
        job={fakeJob({ URL: "" })}
        onDelete={() => {}}
        onClose={() => {}}
      />
    );
    expect(
      screen.queryByRole("button", { name: /open in browser/i })
    ).not.toBeInTheDocument();
  });

  it("shows confirm buttons when delete is clicked", async () => {
    render(<JobDetail job={fakeJob()} onDelete={() => {}} onClose={() => {}} />);

    await userEvent.click(screen.getByRole("button", { name: /delete/i }));

    expect(
      screen.getByRole("button", { name: /confirm delete/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /cancel delete/i })
    ).toBeInTheDocument();
  });

  it("calls onDelete when delete is confirmed", async () => {
    const onDelete = vi.fn();

    render(<JobDetail job={fakeJob()} onDelete={onDelete} onClose={() => {}} />);

    await userEvent.click(screen.getByRole("button", { name: /delete/i }));
    await userEvent.click(
      screen.getByRole("button", { name: /confirm delete/i })
    );
    expect(onDelete).toHaveBeenCalledWith("j1");
  });

  it("does not call onDelete when delete is cancelled", async () => {
    const onDelete = vi.fn();

    render(<JobDetail job={fakeJob()} onDelete={onDelete} onClose={() => {}} />);

    await userEvent.click(screen.getByRole("button", { name: /delete/i }));
    await userEvent.click(
      screen.getByRole("button", { name: /cancel delete/i })
    );
    expect(onDelete).not.toHaveBeenCalled();
    // Confirm buttons should be hidden again.
    expect(
      screen.queryByRole("button", { name: /confirm delete/i })
    ).not.toBeInTheDocument();
  });

  it("calls onClose when close button is clicked", async () => {
    const onClose = vi.fn();
    render(<JobDetail job={fakeJob()} onDelete={() => {}} onClose={onClose} />);

    await userEvent.click(screen.getByRole("button", { name: /close/i }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("shows source and dates", () => {
    render(<JobDetail job={fakeJob()} onDelete={() => {}} onClose={() => {}} />);
    expect(screen.getByText(/reed_uk/)).toBeInTheDocument();
    expect(screen.getByText(/Posted:/)).toBeInTheDocument();
    expect(screen.getByText(/Found:/)).toBeInTheDocument();
  });
});
