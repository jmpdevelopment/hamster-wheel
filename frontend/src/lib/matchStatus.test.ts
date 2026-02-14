import { describe, it, expect } from "vitest";
import {
  buildMatchStatusMeta,
  readMatchScore,
  readMatchStatus,
  readMatchSummary,
} from "./matchStatus";

describe("matchStatus", () => {
  it("maps matched score to percent badge label", () => {
    const meta = buildMatchStatusMeta("matched", 0.81);
    expect(meta.badgeLabel).toBe("Match Score: 81%");
    expect(meta.detailHeadline).toBe("Match Score: 81%");
    expect(meta.badgeVariantClass).toBe("hw-match-badge--matched");
  });

  it("returns unknown status for unsupported values", () => {
    const status = readMatchStatus({ MatchStatus: "mystery" });
    expect(status).toBe("unknown");
  });

  it("parses valid score and rejects out-of-range score", () => {
    expect(readMatchScore({ MatchScore: 0.42 })).toBe(0.42);
    expect(readMatchScore({ MatchScore: 2.5 })).toBeNull();
  });

  it("trims match summary", () => {
    expect(readMatchSummary({ MatchSummary: "  useful summary  " })).toBe(
      "useful summary"
    );
  });
});
