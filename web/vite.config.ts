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
            headers: process.env.JAVA_PROFILER_UI_TOKEN
              ? {
                  Authorization: `Bearer ${process.env.JAVA_PROFILER_UI_TOKEN}`,
                }
              : undefined,
          },
        }
      : undefined,
  },
  test: {
    environment: "jsdom",
    setupFiles: "./src/test/setup.ts",
    include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
    globals: true,
  },
});
