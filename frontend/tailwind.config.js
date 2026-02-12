/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        hw: {
          bg: "rgb(var(--hw-bg) / <alpha-value>)",
          surface: "rgb(var(--hw-surface) / <alpha-value>)",
          "surface-hover": "rgb(var(--hw-surface-hover) / <alpha-value>)",
          border: "rgb(var(--hw-border) / <alpha-value>)",
          text: "rgb(var(--hw-text) / <alpha-value>)",
          "text-muted": "rgb(var(--hw-text-muted) / <alpha-value>)",
          accent: "rgb(var(--hw-accent) / <alpha-value>)",
          "accent-hover": "rgb(var(--hw-accent-hover) / <alpha-value>)",
          danger: "rgb(var(--hw-danger) / <alpha-value>)",
          success: "rgb(var(--hw-success) / <alpha-value>)",
        },
      },
      fontFamily: {
        sans: [
          "Nunito",
          "-apple-system",
          "BlinkMacSystemFont",
          "Segoe UI",
          "Roboto",
          "sans-serif",
        ],
      },
    },
  },
  plugins: [],
}
