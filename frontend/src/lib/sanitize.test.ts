import { describe, it, expect } from "vitest";
import { containsHTML, sanitizeHTML } from "./sanitize";

describe("containsHTML", () => {
  it("returns true for strings with HTML tags", () => {
    expect(containsHTML("<p>Hello</p>")).toBe(true);
    expect(containsHTML("Text <strong>bold</strong> text")).toBe(true);
    expect(containsHTML("<br/>")).toBe(true);
    expect(containsHTML("<br />")).toBe(true);
  });

  it("returns false for plain text", () => {
    expect(containsHTML("Hello world")).toBe(false);
    expect(containsHTML("a < b and c > d")).toBe(false);
    expect(containsHTML("")).toBe(false);
  });
});

describe("sanitizeHTML", () => {
  it("preserves safe formatting tags", () => {
    const html = "<p>Hello <strong>world</strong></p>";
    const result = sanitizeHTML(html);
    expect(result).toContain("<p>");
    expect(result).toContain("<strong>");
    expect(result).toContain("Hello");
  });

  it("removes script tags", () => {
    const html = '<p>Safe</p><script>alert("xss")</script>';
    const result = sanitizeHTML(html);
    expect(result).toContain("Safe");
    expect(result).not.toContain("script");
    expect(result).not.toContain("alert");
  });

  it("removes iframe tags", () => {
    const html = '<p>Content</p><iframe src="evil.com"></iframe>';
    const result = sanitizeHTML(html);
    expect(result).toContain("Content");
    expect(result).not.toContain("iframe");
  });

  it("removes event handler attributes", () => {
    const html = '<p onclick="alert(1)">Click me</p>';
    const result = sanitizeHTML(html);
    expect(result).toContain("Click me");
    expect(result).not.toContain("onclick");
  });

  it("removes javascript: URLs from href", () => {
    const html = '<a href="javascript:alert(1)">Link</a>';
    const result = sanitizeHTML(html);
    expect(result).toContain("Link");
    expect(result).not.toContain("javascript:");
  });

  it("preserves safe href links", () => {
    const html = '<a href="https://example.com">Link</a>';
    const result = sanitizeHTML(html);
    expect(result).toContain('href="https://example.com"');
  });

  it("removes style elements and inline style attributes", () => {
    const html =
      '<style>body{display:none}</style><p style="color:red">Text</p>';
    const result = sanitizeHTML(html);
    expect(result).not.toContain("<style>");
    expect(result).not.toContain("color:red");
    expect(result).toContain("Text");
  });

  it("handles Reed-style job description HTML", () => {
    const html =
      "<p>Kennedys is looking for a Developer<strong></strong>to join our IT team.</p>" +
      "<ul><li>Experience with Go</li><li>Experience with React</li></ul>";
    const result = sanitizeHTML(html);
    expect(result).toContain("<p>");
    expect(result).toContain("<ul>");
    expect(result).toContain("<li>");
    expect(result).toContain("Experience with Go");
  });
});
