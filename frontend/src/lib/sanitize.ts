/**
 * Returns true if the string contains HTML tags.
 */
export function containsHTML(text: string): boolean {
  return /<\/?[a-z][\s\S]*?>/i.test(text);
}

/**
 * Sanitizes HTML by removing dangerous elements and attributes.
 * Uses the browser's built-in DOMParser — no external dependency needed.
 */
export function sanitizeHTML(html: string): string {
  const doc = new DOMParser().parseFromString(html, "text/html");

  // Remove dangerous elements.
  doc
    .querySelectorAll(
      "script, style, iframe, object, embed, form, input, textarea, button, link, meta"
    )
    .forEach((el) => el.remove());

  // Remove event handler attributes and javascript: URLs.
  doc.querySelectorAll("*").forEach((el) => {
    for (const attr of Array.from(el.attributes)) {
      if (
        attr.name.startsWith("on") ||
        (attr.name === "href" &&
          attr.value.trim().toLowerCase().startsWith("javascript:")) ||
        (attr.name === "src" &&
          attr.value.trim().toLowerCase().startsWith("javascript:"))
      ) {
        el.removeAttribute(attr.name);
      }
    }
  });

  return doc.body.innerHTML;
}
