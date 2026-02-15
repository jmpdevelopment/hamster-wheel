const allowedExternalProtocols = new Set(["http:", "https:"]);

// toSafeExternalURL returns a normalized URL string only for http/https URLs.
// Any malformed or non-web URL returns an empty string.
export function toSafeExternalURL(raw: string): string {
  const candidate = raw.trim();
  if (candidate === "") {
    return "";
  }

  let parsed: URL;
  try {
    parsed = new URL(candidate);
  } catch {
    return "";
  }

  if (!allowedExternalProtocols.has(parsed.protocol)) {
    return "";
  }

  return parsed.toString();
}
