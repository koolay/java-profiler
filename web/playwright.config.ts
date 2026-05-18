import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  use: {
    baseURL: "http://127.0.0.1:5175",
    video: process.env.REAL_ACCEPTANCE ? "on" : "retain-on-failure"
  }
});
