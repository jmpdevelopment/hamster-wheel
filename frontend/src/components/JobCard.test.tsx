import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { JobCard } from "./JobCard";

const fakeJob = (overrides = {}) => ({
  ID: "j1",
  Source: "reed_uk",
  SourceID: "src-j1",
  Title: "Senior Go Developer",
  Company: "Acme Corp",
  Location: "London",
  Description: "A great job",
  URL: "https://example.com/job",
  PostedAt: "2026-02-08T10:00:00Z",
  DiscoveredAt: "2026-02-08T11:00:00Z",
  FilterID: "f1",
  IsFavorite: false,
  ...overrides,
});

const defaultProps = {
  job: fakeJob(),
  isSelected: false,
  isChecked: false,
  isFavorite: false,
  onClick: () => {},
  onToggleChecked: () => {},
  onToggleFavorite: () => {},
};

describe("JobCard", () => {
  it("renders job title", () => {
    render(<JobCard {...defaultProps} />);
    expect(screen.getByText("Senior Go Developer")).toBeInTheDocument();
  });

  it("renders company and location", () => {
    render(<JobCard {...defaultProps} />);
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    expect(screen.getByText("London")).toBeInTheDocument();
  });

  it("renders source", () => {
    render(<JobCard {...defaultProps} />);
    expect(screen.getByText("reed_uk")).toBeInTheDocument();
  });

  it("handles missing company", () => {
    render(<JobCard {...defaultProps} job={fakeJob({ Company: "" })} />);
    expect(screen.getByText("London")).toBeInTheDocument();
  });

  it("handles missing location", () => {
    render(<JobCard {...defaultProps} job={fakeJob({ Location: "" })} />);
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
  });

  it("calls onClick when job content is clicked", async () => {
    const onClick = vi.fn();
    render(<JobCard {...defaultProps} onClick={onClick} />);

    await userEvent.click(screen.getByText("Senior Go Developer"));
    expect(onClick).toHaveBeenCalledWith(false);
  });

  it("passes shiftKey when job content is shift-clicked", () => {
    const onClick = vi.fn();
    render(<JobCard {...defaultProps} onClick={onClick} />);

    fireEvent.click(screen.getByText("Senior Go Developer"), {
      shiftKey: true,
    });

    expect(onClick).toHaveBeenCalledWith(true);
  });

  it("calls onToggleChecked when checkbox changes", async () => {
    const onToggleChecked = vi.fn();
    render(<JobCard {...defaultProps} onToggleChecked={onToggleChecked} />);

    await userEvent.click(screen.getByRole("checkbox", { name: /select job/i }));
    expect(onToggleChecked).toHaveBeenCalledWith(true, false);
  });

  it("passes shiftKey when checkbox is shift-clicked", () => {
    const onToggleChecked = vi.fn();
    render(<JobCard {...defaultProps} onToggleChecked={onToggleChecked} />);

    fireEvent.click(screen.getByRole("checkbox", { name: /select job/i }), {
      shiftKey: true,
    });

    expect(onToggleChecked).toHaveBeenCalledWith(true, true);
  });

  it("calls onToggleFavorite when favorite button is clicked", async () => {
    const onToggleFavorite = vi.fn();
    render(<JobCard {...defaultProps} onToggleFavorite={onToggleFavorite} />);

    await userEvent.click(
      screen.getByRole("button", { name: /add senior go developer to favorites/i })
    );
    expect(onToggleFavorite).toHaveBeenCalledOnce();
  });

  it("shows filled star style when favorite", () => {
    render(<JobCard {...defaultProps} isFavorite={true} />);
    expect(
      screen.getByRole("button", {
        name: /remove senior go developer from favorites/i,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("★")).toBeInTheDocument();
  });

  it("has different styling when selected", () => {
    const { container } = render(<JobCard {...defaultProps} isSelected={true} />);
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.className).toContain("bg-hw-accent/10");
  });

  it("is wrapped in React.memo", () => {
    expect((JobCard as any).$$typeof).toBe(Symbol.for("react.memo"));
  });

  it("applies style prop for virtual scrolling", () => {
    const style = { position: "absolute" as const, top: 0, height: 74 };
    const { container } = render(<JobCard {...defaultProps} style={style} />);
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.style.position).toBe("absolute");
    expect(wrapper.style.height).toBe("74px");
  });

  it("shows pending match badge with accessible label", () => {
    render(
      <JobCard
        {...defaultProps}
        job={fakeJob({ MatchStatus: "pending" })}
      />
    );
    expect(screen.getByText("Match pending")).toBeInTheDocument();
    expect(
      screen.getByLabelText("Match status: pending")
    ).toBeInTheDocument();
  });

  it("shows matched badge when status is matched", () => {
    render(
      <JobCard
        {...defaultProps}
        job={fakeJob({ MatchStatus: "matched" })}
      />
    );
    expect(screen.getByText("Match ready")).toBeInTheDocument();
  });

  it("shows failed badge when status is failed", () => {
    render(
      <JobCard
        {...defaultProps}
        job={fakeJob({ MatchStatus: "failed" })}
      />
    );
    expect(screen.getByText("Match failed")).toBeInTheDocument();
  });

  it("hides match badge when status is empty", () => {
    render(<JobCard {...defaultProps} />);
    expect(
      screen.queryByLabelText(/match status:/i)
    ).not.toBeInTheDocument();
  });
});
