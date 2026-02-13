import { render, screen } from "@testing-library/react";
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
  ...overrides,
});

describe("JobCard", () => {
  it("renders job title", () => {
    render(<JobCard job={fakeJob()} isSelected={false} onClick={() => {}} />);
    expect(screen.getByText("Senior Go Developer")).toBeInTheDocument();
  });

  it("renders company and location", () => {
    render(<JobCard job={fakeJob()} isSelected={false} onClick={() => {}} />);
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    expect(screen.getByText("London")).toBeInTheDocument();
  });

  it("renders source", () => {
    render(<JobCard job={fakeJob()} isSelected={false} onClick={() => {}} />);
    expect(screen.getByText("reed_uk")).toBeInTheDocument();
  });

  it("handles missing company", () => {
    render(
      <JobCard
        job={fakeJob({ Company: "" })}
        isSelected={false}
        onClick={() => {}}
      />
    );
    expect(screen.getByText("London")).toBeInTheDocument();
  });

  it("handles missing location", () => {
    render(
      <JobCard
        job={fakeJob({ Location: "" })}
        isSelected={false}
        onClick={() => {}}
      />
    );
    expect(screen.getByText("Acme Corp")).toBeInTheDocument();
  });

  it("calls onClick when clicked", async () => {
    const onClick = vi.fn();
    render(<JobCard job={fakeJob()} isSelected={false} onClick={onClick} />);

    await userEvent.click(screen.getByText("Senior Go Developer"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("has different styling when selected", () => {
    const { container } = render(
      <JobCard job={fakeJob()} isSelected={true} onClick={() => {}} />
    );
    const button = container.querySelector("button");
    expect(button?.className).toContain("bg-hw-accent/10");
  });

  it("is wrapped in React.memo", () => {
    expect((JobCard as any).$$typeof).toBe(Symbol.for("react.memo"));
  });

  it("applies style prop for virtual scrolling", () => {
    const style = { position: "absolute" as const, top: 0, height: 74 };
    const { container } = render(
      <JobCard
        job={fakeJob()}
        isSelected={false}
        onClick={() => {}}
        style={style}
      />
    );
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.style.position).toBe("absolute");
    expect(wrapper.style.height).toBe("74px");
  });
});
