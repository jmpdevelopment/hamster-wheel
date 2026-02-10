import DOMPurify from "dompurify";

/**
 * Returns true if the string contains HTML tags.
 */
export function containsHTML(text: string): boolean {
  return /<\/?[a-z][\s\S]*?>/i.test(text);
}

/**
 * Sanitizes HTML using DOMPurify (allowlist-based).
 * Only permits safe formatting tags — strips scripts, iframes, forms,
 * event handlers, javascript: URLs, and all other dangerous content.
 */
export function sanitizeHTML(html: string): string {
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: [
      "p", "br", "b", "i", "em", "strong", "u", "s",
      "h1", "h2", "h3", "h4", "h5", "h6",
      "ul", "ol", "li",
      "a", "span", "div",
      "table", "thead", "tbody", "tr", "th", "td",
      "blockquote", "pre", "code", "hr", "sub", "sup",
    ],
    ALLOWED_ATTR: ["href", "target", "rel"],
    ALLOW_DATA_ATTR: false,
  });
}
