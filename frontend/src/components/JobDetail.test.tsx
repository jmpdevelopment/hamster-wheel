import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { JobDetail } from "./JobDetail";

// Mock Wails runtime.
vi.mock("@wailsio/runtime", () => ({
  Browser: { OpenURL: vi.fn() },
  Dialogs: { SaveFile: vi.fn().mockResolvedValue("") },
}));

// Mock jobservice bindings.
vi.mock("../../bindings/hamster-wheel/jobservice", () => ({
  RetryFetchDescription: vi.fn(),
  RecalculateMatchScore: vi.fn(),
}));

import { Browser } from "@wailsio/runtime";
import {
  RecalculateMatchScore,
  RetryFetchDescription,
} from "../../bindings/hamster-wheel/jobservice";

const mockedRetry = vi.mocked(RetryFetchDescription);
const mockedRecalculate = vi.mocked(RecalculateMatchScore);

const noop = async () => {};

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
  IsFavorite: false,
  ...overrides,
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe("JobDetail", () => {
  it("renders job title and details", () => {
    render(<JobDetail job={fakeJob()} onDelete={noop} onClose={() => {}} onRefresh={noop} />);

    expect(screen.getByText("Senior Go Developer")).toBeInTheDocument();
    expect(screen.getByText(/Acme Corp/)).toBeInTheDocument();
    expect(screen.getByText(/London/)).toBeInTheDocument();
  });

  it("renders plain text description", () => {
    render(<JobDetail job={fakeJob()} onDelete={noop} onClose={() => {}} onRefresh={noop} />);
    expect(screen.getByText(/Build backend services in Go/)).toBeInTheDocument();
  });

  it("renders HTML description as formatted content", () => {
    render(
      <JobDetail
        job={fakeJob({
          Description: "<p>Looking for a <strong>Go Developer</strong></p>",
        })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={noop}
      />
    );
    expect(screen.getByText(/Looking for a/)).toBeInTheDocument();
    expect(screen.getByText("Go Developer")).toBeInTheDocument();
  });

  it("shows retry button when description is empty", () => {
    render(
      <JobDetail
        job={fakeJob({ Description: "" })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={noop}
      />
    );
    expect(screen.getByText("Description couldn't be loaded.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });

  it("does not show retry button when description is present", () => {
    render(<JobDetail job={fakeJob()} onDelete={noop} onClose={() => {}} onRefresh={noop} />);
    expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument();
  });

  it("calls RetryFetchDescription and onRefresh on successful retry", async () => {
    const onRefresh = vi.fn().mockResolvedValue(undefined);
    mockedRetry.mockResolvedValue(undefined);

    render(
      <JobDetail
        job={fakeJob({ Description: "" })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={onRefresh}
      />
    );

    await userEvent.click(screen.getByRole("button", { name: /retry/i }));

    await waitFor(() => {
      expect(mockedRetry).toHaveBeenCalledWith("j1");
      expect(onRefresh).toHaveBeenCalledOnce();
    });
  });

  it("shows error message when retry fails", async () => {
    mockedRetry.mockRejectedValue(new Error("Network timeout"));

    render(
      <JobDetail
        job={fakeJob({ Description: "" })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={noop}
      />
    );

    await userEvent.click(screen.getByRole("button", { name: /retry/i }));

    await waitFor(() => {
      expect(screen.getByText("Network timeout")).toBeInTheDocument();
    });
  });

  it("calls BrowserOpenURL when Open in Browser is clicked", async () => {
    render(<JobDetail job={fakeJob()} onDelete={noop} onClose={() => {}} onRefresh={noop} />);

    await userEvent.click(
      screen.getByRole("button", { name: /open in browser/i })
    );
    expect(Browser.OpenURL).toHaveBeenCalledWith("https://example.com/job");
  });

  it("hides Open in Browser when URL is empty", () => {
    render(
      <JobDetail
        job={fakeJob({ URL: "" })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={noop}
      />
    );
    expect(
      screen.queryByRole("button", { name: /open in browser/i })
    ).not.toBeInTheDocument();
  });

  it("shows confirm buttons when delete is clicked", async () => {
    render(<JobDetail job={fakeJob()} onDelete={noop} onClose={() => {}} onRefresh={noop} />);

    await userEvent.click(screen.getByRole("button", { name: /delete/i }));

    expect(
      screen.getByRole("button", { name: /confirm delete/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /cancel delete/i })
    ).toBeInTheDocument();
  });

  it("calls onDelete when delete is confirmed", async () => {
    const onDelete = vi.fn().mockResolvedValue(undefined);

    render(<JobDetail job={fakeJob()} onDelete={onDelete} onClose={() => {}} onRefresh={noop} />);

    await userEvent.click(screen.getByRole("button", { name: /delete/i }));
    await userEvent.click(
      screen.getByRole("button", { name: /confirm delete/i })
    );
    expect(onDelete).toHaveBeenCalledWith("j1");
  });

  it("does not call onDelete when delete is cancelled", async () => {
    const onDelete = vi.fn().mockResolvedValue(undefined);

    render(<JobDetail job={fakeJob()} onDelete={onDelete} onClose={() => {}} onRefresh={noop} />);

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
    render(<JobDetail job={fakeJob()} onDelete={noop} onClose={onClose} onRefresh={noop} />);

    await userEvent.click(screen.getByRole("button", { name: /close/i }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("shows source and dates", () => {
    render(<JobDetail job={fakeJob()} onDelete={noop} onClose={() => {}} onRefresh={noop} />);
    expect(screen.getByText(/reed_uk/)).toBeInTheDocument();
    expect(screen.getByText(/Posted:/)).toBeInTheDocument();
    expect(screen.getByText(/Found:/)).toBeInTheDocument();
    expect(
      screen.queryByText("Adzuna provides a description snippet, not the full job ad.")
    ).not.toBeInTheDocument();
  });

  it("shows Adzuna attribution source label", () => {
    render(
      <JobDetail
        job={fakeJob({ Source: "adzuna_gb" })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={noop}
      />
    );
    expect(screen.getByText("Jobs by Adzuna")).toBeInTheDocument();
    expect(
      screen.getByText("Adzuna provides a description snippet, not the full job ad.")
    ).toBeInTheDocument();
  });

  it("shows match score headline when status is matched", () => {
    render(
      <JobDetail
        job={fakeJob({ MatchStatus: "matched", MatchScore: 0.84 })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={noop}
      />
    );
    expect(screen.getByText("Match Score: 84%")).toBeInTheDocument();
    const statusBadge = screen.getByText("Matched");
    expect(statusBadge).toBeInTheDocument();
    expect(statusBadge.className).toContain("hw-match-badge");
    expect(statusBadge.className).toContain("hw-match-badge--matched");
  });

  it("shows match summary when available", () => {
    render(
      <JobDetail
        job={fakeJob({
          MatchStatus: "matched",
          MatchScore: 0.65,
          MatchSummary: "Strong alignment with backend API keywords.",
        })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={noop}
      />
    );
    expect(
      screen.getByText("Strong alignment with backend API keywords.")
    ).toBeInTheDocument();
  });

  it("shows provider label in detail summary panel", () => {
    render(
      <JobDetail
        job={fakeJob({
          MatchStatus: "matched",
          MatchScore: 0.65,
          MatchSummary: "Provider: heuristic_v1\nStrong alignment with backend API keywords.",
        })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={noop}
      />
    );
    expect(screen.getByText("LLM provider: Heuristic (Local)")).toBeInTheDocument();
    expect(
      screen.getByText("Strong alignment with backend API keywords.")
    ).toBeInTheDocument();
  });

  it("calls RecalculateMatchScore and refreshes", async () => {
    const onRefresh = vi.fn().mockResolvedValue(undefined);
    mockedRecalculate.mockResolvedValue(undefined);

    render(
      <JobDetail
        job={fakeJob({ MatchStatus: "matched", MatchScore: 0.5 })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={onRefresh}
      />
    );

    await userEvent.click(
      screen.getByRole("button", { name: /recalculate score/i })
    );

    await waitFor(() => {
      expect(mockedRecalculate).toHaveBeenCalledWith("j1");
      expect(onRefresh).toHaveBeenCalledOnce();
      expect(screen.getByText("Recalculation queued.")).toBeInTheDocument();
    });
  });

  it("disables recalculate button while processing", () => {
    render(
      <JobDetail
        job={fakeJob({ MatchStatus: "processing" })}
        onDelete={noop}
        onClose={() => {}}
        onRefresh={noop}
      />
    );

    expect(
      screen.getByRole("button", { name: /calculating/i })
    ).toBeDisabled();
  });
});
