import { describe, expect, it } from "vitest";

import { toSafeExternalURL } from "./url";

describe("toSafeExternalURL", () => {
  it("accepts https URLs", () => {
    expect(toSafeExternalURL("https://example.com/job")).toBe(
      "https://example.com/job"
    );
  });

  it("accepts http URLs", () => {
    expect(toSafeExternalURL("http://example.com/job")).toBe(
      "http://example.com/job"
    );
  });

  it("rejects non-http schemes", () => {
    expect(toSafeExternalURL("javascript:alert(1)")).toBe("");
    expect(toSafeExternalURL("file:///tmp/test")).toBe("");
  });

  it("rejects malformed URLs", () => {
    expect(toSafeExternalURL("not a url")).toBe("");
    expect(toSafeExternalURL("")).toBe("");
  });
});
