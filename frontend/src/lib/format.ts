/**
 * Date formatting utilities for displaying Go time.Time values.
 * Wails serializes Go time.Time as ISO 8601 strings (e.g., "2026-02-08T10:30:00Z").
 * These may also be null/undefined for optional fields like PostedAt.
 */

/**
 * Returns a human-readable relative time string (e.g., "2h ago", "3d ago").
 * Returns "—" for null/undefined/invalid dates.
 */
export function relativeTime(date: unknown): string {
  const ms = parseDate(date);
  if (ms === null) return "—";

  const seconds = Math.floor((Date.now() - ms) / 1000);
  if (seconds < 0) return "just now";
  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 2592000) return `${Math.floor(seconds / 86400)}d ago`;
  return formatDate(date);
}

/**
 * Returns a formatted date string (e.g., "8 Feb 2026").
 * Returns "—" for null/undefined/invalid dates.
 */
export function formatDate(date: unknown): string {
  const ms = parseDate(date);
  if (ms === null) return "—";

  return new Date(ms).toLocaleDateString("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

/**
 * Parses a date value from Wails (Go time.Time → ISO string or null)
 * into a Unix timestamp in ms, or null if invalid.
 */
function parseDate(date: unknown): number | null {
  if (date === null || date === undefined) return null;
  if (typeof date === "string") {
    const ms = Date.parse(date);
    return isNaN(ms) ? null : ms;
  }
  if (typeof date === "number") return date;
  return null;
}
