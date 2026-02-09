import { describe, it, expect, vi, afterEach } from "vitest";
import { relativeTime, formatDate } from "./format";

describe("formatDate", () => {
  it("formats a valid ISO date string", () => {
    expect(formatDate("2026-02-08T10:30:00Z")).toBe("8 Feb 2026");
  });

  it("returns dash for null", () => {
    expect(formatDate(null)).toBe("—");
  });

  it("returns dash for undefined", () => {
    expect(formatDate(undefined)).toBe("—");
  });

  it("returns dash for empty string", () => {
    expect(formatDate("")).toBe("—");
  });

  it("returns dash for invalid date string", () => {
    expect(formatDate("not-a-date")).toBe("—");
  });

  it("handles a numeric timestamp", () => {
    const ms = new Date("2026-01-15T00:00:00Z").getTime();
    expect(formatDate(ms)).toBe("15 Jan 2026");
  });

  it("returns dash for non-date object", () => {
    expect(formatDate({ foo: "bar" })).toBe("—");
  });
});

describe("relativeTime", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns 'just now' for dates less than a minute ago", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-02-08T12:00:30Z"));
    expect(relativeTime("2026-02-08T12:00:00Z")).toBe("just now");
  });

  it("returns minutes ago", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-02-08T12:05:00Z"));
    expect(relativeTime("2026-02-08T12:00:00Z")).toBe("5m ago");
  });

  it("returns hours ago", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-02-08T15:00:00Z"));
    expect(relativeTime("2026-02-08T12:00:00Z")).toBe("3h ago");
  });

  it("returns days ago", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-02-11T12:00:00Z"));
    expect(relativeTime("2026-02-08T12:00:00Z")).toBe("3d ago");
  });

  it("returns formatted date for old dates (>30 days)", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-01T12:00:00Z"));
    expect(relativeTime("2026-02-08T12:00:00Z")).toBe("8 Feb 2026");
  });

  it("returns 'just now' for future dates", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-02-08T12:00:00Z"));
    expect(relativeTime("2026-02-08T13:00:00Z")).toBe("just now");
  });

  it("returns dash for null", () => {
    expect(relativeTime(null)).toBe("—");
  });

  it("returns dash for undefined", () => {
    expect(relativeTime(undefined)).toBe("—");
  });

  it("returns dash for invalid string", () => {
    expect(relativeTime("garbage")).toBe("—");
  });
});
