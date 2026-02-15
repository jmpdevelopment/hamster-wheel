import { describe, expect, it } from "vitest";
import {
  ADZUNA_ATTRIBUTION_LABEL,
  ADZUNA_ATTRIBUTION_URL,
  ADZUNA_SNIPPET_NOTICE,
  ADZUNA_SOURCE,
  REED_SOURCE,
  listSourceOptions,
  sourceAttributionURL,
  sourceDescriptionNotice,
  sourceDisplayLabel,
} from "./jobSource";

describe("jobSource", () => {
  it("returns static source options", () => {
    expect(listSourceOptions()).toEqual([
      { value: REED_SOURCE, label: "Reed UK (reed_uk)" },
      {
        value: ADZUNA_SOURCE,
        label: "Adzuna UK (adzuna_gb, snippet descriptions)",
      },
    ]);
  });

  it("uses Adzuna attribution label for adzuna source", () => {
    expect(sourceDisplayLabel(ADZUNA_SOURCE)).toBe(ADZUNA_ATTRIBUTION_LABEL);
    expect(sourceDisplayLabel("ADZUNA_GB")).toBe(ADZUNA_ATTRIBUTION_LABEL);
  });

  it("returns original source label for non-adzuna sources", () => {
    expect(sourceDisplayLabel(REED_SOURCE)).toBe(REED_SOURCE);
    expect(sourceDisplayLabel("custom_source")).toBe("custom_source");
  });

  it("provides Adzuna attribution URL only for adzuna source", () => {
    expect(sourceAttributionURL(ADZUNA_SOURCE)).toBe(ADZUNA_ATTRIBUTION_URL);
    expect(sourceAttributionURL(REED_SOURCE)).toBe("");
  });

  it("returns description notice only for adzuna source", () => {
    expect(sourceDescriptionNotice(ADZUNA_SOURCE)).toBe(ADZUNA_SNIPPET_NOTICE);
    expect(sourceDescriptionNotice("ADZUNA_GB")).toBe(ADZUNA_SNIPPET_NOTICE);
    expect(sourceDescriptionNotice(REED_SOURCE)).toBe("");
  });
});
