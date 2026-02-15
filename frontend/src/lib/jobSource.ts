export const REED_SOURCE = "reed_uk";
export const ADZUNA_SOURCE = "adzuna_gb";

export const ADZUNA_ATTRIBUTION_LABEL = "Jobs by Adzuna";
export const ADZUNA_ATTRIBUTION_URL = "https://www.adzuna.co.uk";
export const ADZUNA_SNIPPET_NOTICE =
  "Adzuna provides a description snippet, not the full job ad.";

export interface SourceOption {
  value: string;
  label: string;
}

const sourceOptions: SourceOption[] = [
  { value: REED_SOURCE, label: "Reed UK (reed_uk)" },
  { value: ADZUNA_SOURCE, label: "Adzuna UK (adzuna_gb, snippet descriptions)" },
];

function normalizeSource(source: string): string {
  return source.trim().toLowerCase();
}

export function listSourceOptions(): SourceOption[] {
  return sourceOptions;
}

export function sourceDisplayLabel(source: string): string {
  switch (normalizeSource(source)) {
    case ADZUNA_SOURCE:
      return ADZUNA_ATTRIBUTION_LABEL;
    default:
      return source;
  }
}

export function sourceAttributionURL(source: string): string {
  switch (normalizeSource(source)) {
    case ADZUNA_SOURCE:
      return ADZUNA_ATTRIBUTION_URL;
    default:
      return "";
  }
}

export function sourceDescriptionNotice(source: string): string {
  switch (normalizeSource(source)) {
    case ADZUNA_SOURCE:
      return ADZUNA_SNIPPET_NOTICE;
    default:
      return "";
  }
}
