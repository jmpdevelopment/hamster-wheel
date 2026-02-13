import { useState, useEffect, useCallback } from "react";
import {
  GetTheme,
  SetTheme,
} from "../../bindings/hamster-wheel/settingsservice";

export type ThemePreference = "dark" | "light" | "system";
type ResolvedTheme = "dark" | "light";

export interface UseThemeReturn {
  theme: ThemePreference;
  setTheme: (theme: ThemePreference) => Promise<void>;
}

function applyTheme(resolved: ResolvedTheme): void {
  const html = document.documentElement;
  if (resolved === "light") {
    html.classList.add("light");
    html.classList.remove("dark");
  } else {
    html.classList.remove("light");
    html.classList.add("dark");
  }
}

function getSystemTheme(): ResolvedTheme {
  return window.matchMedia("(prefers-color-scheme: light)").matches
    ? "light"
    : "dark";
}

function resolveTheme(preference: ThemePreference): ResolvedTheme {
  if (preference === "system") {
    return getSystemTheme();
  }
  return preference;
}

export function useTheme(): UseThemeReturn {
  const [theme, setThemeState] = useState<ThemePreference>("system");

  // Load saved theme on mount.
  useEffect(() => {
    GetTheme()
      .then((saved) => {
        const preference: ThemePreference =
          saved === "dark" || saved === "light" || saved === "system"
            ? saved
            : "system";
        setThemeState(preference);
        applyTheme(resolveTheme(preference));
      })
      .catch((err: unknown) => {
        console.error("Failed to load theme:", err);
        applyTheme(resolveTheme("system"));
      });
  }, []);

  // Apply theme whenever the preference changes.
  useEffect(() => {
    applyTheme(resolveTheme(theme));
  }, [theme]);

  // Listen for system theme changes when preference is "system".
  useEffect(() => {
    if (theme !== "system") return;

    const mql = window.matchMedia("(prefers-color-scheme: light)");
    const handler = () => applyTheme(getSystemTheme());
    mql.addEventListener("change", handler);
    return () => mql.removeEventListener("change", handler);
  }, [theme]);

  const handleSetTheme = useCallback(async (newTheme: ThemePreference) => {
    setThemeState(newTheme);
    applyTheme(resolveTheme(newTheme));
    try {
      await SetTheme(newTheme);
    } catch (err: unknown) {
      console.error("Failed to save theme:", err);
    }
  }, []);

  return { theme, setTheme: handleSetTheme };
}
