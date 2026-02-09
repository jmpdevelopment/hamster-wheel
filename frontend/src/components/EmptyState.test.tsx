import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { EmptyState } from "./EmptyState";

describe("EmptyState", () => {
  it("renders title and description", () => {
    render(<EmptyState title="No jobs" description="Create a filter to start." />);

    expect(screen.getByText("No jobs")).toBeInTheDocument();
    expect(screen.getByText("Create a filter to start.")).toBeInTheDocument();
  });
});
