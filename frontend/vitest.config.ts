import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
    coverage: {
      provider: "custom",
      customProviderModule: "./tools/vitest-lite-coverage-provider.mjs",
      reportsDirectory: "./coverage",
      reporter: ["text-summary"],
      exclude: [
        "**/*.test.*",
        "**/*.spec.*",
        "src/test-setup.ts",
        "bindings/**",
        "wailsjs/**",
      ],
    },
  },
});
