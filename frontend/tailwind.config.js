/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        hw: {
          bg: "#1b2636",
          surface: "#243044",
          "surface-hover": "#2d3d55",
          border: "#374357",
          text: "#e2e8f0",
          "text-muted": "#94a3b8",
          accent: "#f59e0b",
          "accent-hover": "#d97706",
          danger: "#ef4444",
          success: "#22c55e",
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

