import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: process.env.VITE_DEV_BACKEND_URL
      ? {
          "/api": {
            target: process.env.VITE_DEV_BACKEND_URL,
            changeOrigin: true,
          },
        }
      : undefined,
  },
  test: {
    environment: "jsdom",
    setupFiles: "./src/test/setup.ts",
    include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
    globals: true
  }
});
